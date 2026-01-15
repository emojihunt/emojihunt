import type {
  AblyWorkerMessage, PresenceMessage, ConnectionState, StatusMessage,
  SyncMessage, User,
} from './utils/types';

let state: ConnectionState = "disconnected";
const updateState = (s: ConnectionState) =>
  (state = s, broadcast({ event: "_", state }));

const rewind = new Array<SyncMessage>();
const activity = new Map<string, [number, boolean]>();
let activityChanged = false;

let latestPresence: PresenceMessage | undefined;
const sheets = new Map<string, number>();
const users = new Map<string, User>();

let backoff = 500;

// Keep a list of connected clients.
//
// HACK: detect when a client disconnects using the Web Locks API. The client
// generates a UUID, takes a lock with that ID, then sends us the ID. When we
// succeed in acquiring the same lock, the client must be dead.
//
type AbstractPort = MessagePort | DedicatedWorkerGlobalScope;
const ports: Map<string, [AbstractPort, number | null]> = new Map();
const unicast = (p: AbstractPort, m: AblyWorkerMessage) => p.postMessage(m);
const broadcast = (m: AblyWorkerMessage) => ports.forEach(([p, _]) => p.postMessage(m));
const broadcastHome = (m: AblyWorkerMessage) => (
  ports.forEach(([p, z]) => (z === null) && p.postMessage(m))
);
const broadcastPuzzles = (m: AblyWorkerMessage) => (
  ports.forEach(([p, z]) => (z !== null) && p.postMessage(m))
);

self.addEventListener("connect", (e: any) => {
  for (const port of e.ports) {
    port.addEventListener("message", (e: MessageEvent<StatusMessage>) =>
      handleStatusMessage(e, port));
    port.start();
  }
});

// Fallback: can be run as a non-shared Worker, for Chrome for Android.
self.addEventListener("message", (e: MessageEvent<StatusMessage>) =>
  handleStatusMessage(e, null)
);

const handleStatusMessage = (e: MessageEvent<StatusMessage>, port_: MessagePort | null) => {
  const { event, id, puzzle } = e.data;
  switch (event) {
    case "start":
      let port: AbstractPort;
      if (port_) {
        console.log(`[${id}] Joined`);
        navigator.locks.request(id, () => {
          ports.delete(id);
          if (activity.delete(id)) {
            activityChanged = true;
          }
          console.log(`[${id}] Left`);
        });
        port = port_;
      } else { // dedicated worker fallback
        console.log("Worker launched in dedicated scope");
        port = self as DedicatedWorkerGlobalScope;
      }
      ports.set(id, [port, e.data.puzzle]);
      unicast(port, { event: "_", state });
      rewind.forEach(data => unicast(port, { event: "sync", data }));
      unicast(port, {
        event: "users", data: {
          users: Object.fromEntries(
            users.entries().map(([s, u]) => [s, [u.username, u.avatarUrl]])
          ),
          replace: true,
        }
      });
      if (!puzzle) {
        if (latestPresence) {
          unicast(port, { event: "presence", data: latestPresence });
        }
        unicast(port, {
          event: "sheets", data: {
            sheets: Object.fromEntries(sheets.entries()),
            replace: true,
          }
        });
      }
      break;
    case "activity":
      const { active } = e.data;
      console.log(`[${id}]`, active ? "Active" : "Inactive", `(Puzzle #${puzzle})`);
      activity.set(id, [puzzle, active]);
      activityChanged = true;
      break;
    default:
      console.error("Unexpected status event:", e.data);
  }
};

const reportActivity = (socket: WebSocket) => {
  const computed = new Map<number, boolean>();
  activity.forEach(([puzzle, active]) =>
    computed.set(puzzle, computed.get(puzzle) || active)
  );
  const raw = Object.fromEntries(computed);
  console.log("[rx] Activity", raw);
  socket.send(JSON.stringify({ event: "activity", activity: raw }));
  activityChanged = false;
};
setInterval(() => (activityChanged && socket && reportActivity(socket)), 5_000);

let socket: WebSocket | null = null;
const reconnect = () => new Promise((resolve) => {
  let endpoint = origin.includes("localhost") // can't useAppConfig() here :(
    ? "ws://localhost:9090/rx"
    : "wss://live.emojihunt.org/rx";
  const last = rewind.at(-1);
  if (last) {
    endpoint = `${endpoint}?after=${last.change_id}`;
  }
  const ws = new WebSocket(endpoint);
  const timer = setTimeout(() => (ws.close(), resolve(null)), backoff);
  ws.addEventListener("open", () => {
    console.log("[rx] ...connected!");
    updateState("connected");
    clearTimeout(timer);
    socket = ws;
    if (activity.size > 0) {
      reportActivity(ws);
    }
  });
  ws.addEventListener("close", (e) => resolve(e));
  ws.addEventListener("error", (e) => resolve(e));

  ws.addEventListener("message", (e) => {
    const msg = JSON.parse(e.data) as AblyWorkerMessage;
    backoff = 500; // only reset after first successful message
    switch (msg.event) {
      case "_":
        console.error("[*] Invalid", msg);
        break;
      case "m":
        broadcastPuzzles(msg);
        break;
      case "presence":
        console.log("[*] Presence", msg.data);
        latestPresence = msg.data;
        broadcastHome(msg);
        break;
      case "settings":
        console.log("[*] Settings", msg.data);
        broadcast(msg);
        break;
      case "sheets":
        console.log("[*] Sheets", msg.data);
        if (msg.data.replace) sheets.clear();
        Object.entries(msg.data.sheets).forEach(
          ([k, v]) => sheets.set(k, v)
        );
        broadcastHome(msg);
        break;
      case "sync":
        console.log("[*] Sync", msg.data);
        rewind.push(msg.data);
        if (rewind.length >= 256) rewind.shift();
        broadcast(msg);
        break;
      case "users":
        console.log("[*] Users", msg.data);
        if (msg.data.replace) users.clear();
        Object.entries(msg.data.users).forEach(
          ([k, [username, avatarUrl]]) => users.set(k, { username, avatarUrl })
        );
        msg.data.delete?.forEach((k) => users.delete(k));
        broadcast(msg);
        break;
      default:
        ((x: never) => console.warn("[*] Unknown", x))(msg);
    }
  });

  console.log(`[rx] Connecting to ${endpoint}...`);
});

console.log("Live worker initialized");
(async () => {
  while (true) {
    const error = await reconnect();
    socket = null;
    backoff = Math.min(backoff * 2, 4_000);
    updateState("disconnected");
    if (error === null) {
      console.warn("[rx] Timed out");
      // no sleep, we've already been waiting
    } else {
      if (error instanceof CloseEvent) {
        console.warn("[rx] Closed", error.code, error.reason);
        if (error.code === 4004) {
          console.error("[rx] Unrecoverable consistency failure");
          updateState("dead");
          return;
        }
      } else {
        console.error("[rx]", error);
      }
      await new Promise((r) => setTimeout(r, backoff));
    }
  }
})();
