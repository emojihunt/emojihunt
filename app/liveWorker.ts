import type { AblyWorkerMessage, ConnectionState } from './utils/types';

let state: ConnectionState = "disconnected";
const updateState = (s: ConnectionState) =>
  (state = s, broadcast({ event: "client", state }));

const rewind = new Array<AblyWorkerMessage>();

// Keep a list of connected clients.
//
// HACK: detect when a client disconnects using the Web Locks API. The client
// generates a UUID, requests a lock with that ID, then sends us the ID. When
// we succeed in acquiring the same lock, the client must be dead.
//
const ports: Map<string, MessagePort> = new Map();
const broadcast = (m: AblyWorkerMessage) => ports.forEach(p => p.postMessage(m));

self.addEventListener("connect", (e: any) => {
  for (const port of e.ports) {
    port.addEventListener("message", (e: MessageEvent<string>) => {
      const id = e.data;
      console.log(`[${id}] Connected`);
      ports.set(id, port);
      navigator.locks.request(id, () => {
        ports.delete(id);
        console.log(`[${id}] Disconnected`);
      });

      port.postMessage({ event: "client", state });
      rewind.forEach(r => port.postMessage(r));
    });
    port.start();
  }
});

// Fallback: can be run as a non-shared Worker, for Chrome for Android.
self.addEventListener("message", (e: MessageEvent<string>) => {
  if (e.data === "start") {
    console.log("Worker launched in dedicated scope");
    ports.set(crypto.randomUUID(), self as any);
  }
});

const reconnect = () => new Promise((resolve) => {
  const endpoint = origin.includes("localhost") // no useAppConfig() here :(
    ? "ws://localhost:9090/rx"
    : "wss://live.emojihunt.org/rx";
  const ws = new WebSocket(endpoint);
  const timer = setTimeout(() => (ws.close(), resolve(null)), 5_000);
  ws.addEventListener("open", () => {
    console.log("[rx] ...connected!");
    updateState("connected");
    clearTimeout(timer);
    // TODO: make sure rewind window is up to date, else go to dead state
  });
  ws.addEventListener("close", (e) => resolve(e));
  ws.addEventListener("error", (e) => resolve(e));

  ws.addEventListener("message", (e) => {
    const msg = JSON.parse(e.data) as AblyWorkerMessage;
    // TODO: cache settings messages (add change ID on server, maybe include in rewind)
    switch (msg.event) {
      case "sync":
        console.log("[*] Sync", msg.data);
        rewind.push(msg);
        if (rewind.length >= 256) rewind.shift();
        break;
      case "settings":
        console.log("[*] Settings", msg.data);
        break;
    }
    broadcast(msg);
  });

  console.log(`[rx] Connecting to ${endpoint}...`);
});

console.log("Live worker initialized");
while (true) {
  const error = await reconnect();
  if (error === null) {
    console.warn("[rx] Timed out");
  } else if (error instanceof CloseEvent) {
    console.warn("[rx] Closed");
  } else if (error instanceof ErrorEvent) {
    console.error("[rx]", error.error);
  } else {
    console.error("[rx] unknown:", error);
  }
  await new Promise((r) => setTimeout(r, 1_000));
}
