import type { AblyWorkerMessage, ConnectionState } from './utils/types';

let state: ConnectionState = "disconnected";
const rewind = new Array<AblyWorkerMessage>();

// Keep a list of clients as they connect.
// TODO: implement garbage collection with Web Locks
const ports: Array<MessagePort | DedicatedWorkerGlobalScope> = [];
const broadcast = (m: AblyWorkerMessage) => ports.forEach(p => p.postMessage(m));
const updateState = (s: ConnectionState) =>
  (state = s, broadcast({ event: "client", state }));
self.addEventListener("connect", (e: any) => {
  console.log("Client connected");
  for (const port of e.ports) {
    ports.push(port);
    port.postMessage({ event: "client", state });
    rewind.forEach(r => port.postMessage(r));
  }
});

// Fallback: can be run as a non-shared Worker, for Chrome for Android.
self.addEventListener("message", (e: MessageEvent<string>) => {
  if (e.data === "start") {
    console.log("Worker launched in dedicated scope");
    ports.push(self as DedicatedWorkerGlobalScope);
  }
});

const reconnect = () => new Promise((resolve) => {
  const endpoint = origin.includes("localhost") // no useAppConfig() here :(
    ? "ws://localhost:9090/rx"
    : "wss://live.emojihunt.org/rx";
  const ws = new WebSocket(endpoint);
  const timer = setTimeout(() => (ws.close(), resolve(null)), 5_000);
  ws.addEventListener("open", () => {
    console.log("[rx] Connected!");
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
        console.log("Sync", msg.data);
        rewind.push(msg);
        if (rewind.length >= 256) rewind.shift();
        break;
      case "settings":
        console.log("Settings", msg.data);
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
