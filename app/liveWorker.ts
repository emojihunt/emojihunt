import type {
  AblyWorkerMessage, ConnectionState, StatusMessage, SyncMessage,
} from './utils/types';

let state: ConnectionState = "disconnected";
const updateState = (s: ConnectionState) =>
  (state = s, broadcast({ event: "client", state }));

const rewind = new Array<SyncMessage>();
const activity = new Map<string, [number, boolean]>();
let activityChanged = false;

let backoff = 500;

// Keep a list of connected clients.
//
// HACK: detect when a client disconnects using the Web Locks API. The client
// generates a UUID, takes a lock with that ID, then sends us the ID. When we
// succeed in acquiring the same lock, the client must be dead.
//
const ports: Map<string, MessagePort | DedicatedWorkerGlobalScope> = new Map();
const unicast = (
  p: MessagePort | DedicatedWorkerGlobalScope, m: AblyWorkerMessage,
) => p.postMessage(m);
const broadcast = (m: AblyWorkerMessage) => ports.forEach(p => p.postMessage(m));

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

const handleStatusMessage = (e: MessageEvent<StatusMessage>, port: any) => {
  const { event, id } = e.data;
  switch (event) {
    case "start":
      if (port) {
        console.log(`[${id}] Joined`);
        navigator.locks.request(id, () => {
          ports.delete(id);
          activity.delete(id);
          activityChanged = true;
          console.log(`[${id}] Left`);
        });
      } else { // dedicated worker fallback
        console.log("Worker launched in dedicated scope");
        port = self as DedicatedWorkerGlobalScope;
      }
      ports.set(id, port);
      unicast(port, { event: "client", state });
      rewind.forEach(data => unicast(port, { event: "sync", data }));
      break;
    case "activity":
      const { puzzle, active } = e.data;
      console.log(`[${id}]`, active ? "Active" : "Inactive", `(Puzzle #${puzzle})`);
      activity.set(id, [puzzle, active]);
      activityChanged = true;
      break;
    default:
      console.error("Unexpected status event:", e.data);
  }
};

const sendActivity = (socket: WebSocket) => {
  const computed = new Map<number, boolean>();
  activity.forEach(([puzzle, active]) =>
    computed.set(puzzle, computed.get(puzzle) || active)
  );
  const raw = Object.fromEntries(computed);
  console.log("[rx] Activity", raw);
  socket.send(JSON.stringify({ event: "activity", activity: raw }));
  activityChanged = false;
};
setInterval(() => (activityChanged && socket && sendActivity(socket)), 5_000);

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
  });
  ws.addEventListener("close", (e) => resolve(e));
  ws.addEventListener("error", (e) => resolve(e));

  ws.addEventListener("message", (e) => {
    const msg = JSON.parse(e.data) as AblyWorkerMessage;
    backoff = 500; // only reset after first successful message
    switch (msg.event) {
      case "sync":
        console.log("[*] Sync", msg.data);
        rewind.push(msg.data);
        if (rewind.length >= 256) rewind.shift();
        break;
      case "settings":
        console.log("[*] Settings", msg.data);
        break;
      case "m":
        break;
      default:
        console.warn("[*] Unknown", msg);
    }
    broadcast(msg);
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
