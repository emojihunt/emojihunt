import type { AblyWorkerMessage, ConnectionState, SyncMessage } from './utils/types';

let state: ConnectionState = "disconnected";
const updateState = (s: ConnectionState) =>
  (state = s, broadcast({ event: "client", state }));

const rewind = new Array<SyncMessage>();

let backoff = 1_000;

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
    port.addEventListener("message", (e: MessageEvent<string>) => {
      const id = e.data;
      console.log(`[${id}] Joined`);
      ports.set(id, port);
      navigator.locks.request(id, () => {
        ports.delete(id);
        console.log(`[${id}] Left`);
      });
      unicast(port, { event: "client", state });
      rewind.forEach(data => unicast(port, { event: "sync", data }));
    });
    port.start();
  }
});

// Fallback: can be run as a non-shared Worker, for Chrome for Android.
self.addEventListener("message", (e: MessageEvent<string>) => {
  if (e.data === "start") {
    console.log("Worker launched in dedicated scope");
    const port = self as DedicatedWorkerGlobalScope;
    ports.set(crypto.randomUUID(), port);
    unicast(port, { event: "client", state });
    rewind.forEach(data => unicast(port, { event: "sync", data }));
  }
});

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
    backoff = 1_000;
  });
  ws.addEventListener("close", (e) => resolve(e));
  ws.addEventListener("error", (e) => resolve(e));

  ws.addEventListener("message", (e) => {
    const msg = JSON.parse(e.data) as AblyWorkerMessage;
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
    updateState("disconnected");
    backoff = Math.min(backoff * 2, 16_000);
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
