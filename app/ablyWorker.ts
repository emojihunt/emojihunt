import { BaseRealtime, FetchRequest, WebSocketTransport } from 'ably/modular';
import type { AblyWorkerMessage, ConnectionState } from './utils/types';

// Keep a list of clients as they connect. There's no easy way to implement
// garbage collection, unfortunately...
const ports: Array<MessagePort | DedicatedWorkerGlobalScope> = [];
const broadcast = (m: AblyWorkerMessage) => ports.forEach(p => p.postMessage(m));
self.addEventListener("connect", (e: any) => {
  console.log("Client connected");
  for (const port of e.ports) {
    ports.push(port);
    port.postMessage({ event: "_", state });
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

const client = new BaseRealtime({
  authCallback: async (_, callback) => {
    console.log("Fetching Ably token...");
    // We can't use useAppConfig() here :(
    const apiBase = origin.includes("localhost") ?
      "http://localhost:8080" : "https://api.emojihunt.org";
    const r = await fetch(`${apiBase}/ably`, {
      method: "POST",
      credentials: "include",
    });
    if (r.status !== 200) {
      const msg = `HTTP ${r.status}: ${await r.text()}`;
      console.error(msg);
      callback(msg, null);
    } else {
      console.log("Received Ably token");
      callback(null, await r.json());
    }
  },
  plugins: {
    FetchRequest,
    WebSocketTransport,
  },
});

// Enable rewind. There's a brief window where changes would otherwise be
// missed: between when Vercel finishes making the API request to /home during
// server-side rendering and when we subscribe to the channel on the frontend.
//
// Note: Ably's maximum rewind window is 2 minutes or 100 messages.
//
// We also maintain a local 100-message cache for clients that connect later.
//
const huntbot = client.channels.get("huntbot", { params: { rewind: "2m" } });
const rewind = new Array();
let state: ConnectionState = "disconnected";

// Broadcast all messages on the `huntbot` channel to all clients.
huntbot.subscribe("sync", (e: any) => {
  e.event = e.name;
  console.log("Sync", e);
  if (rewind.length >= 100) rewind.shift();
  rewind.push(e);
  broadcast(e);
});
huntbot.subscribe("settings", (e: any) => {
  e.event = e.name;
  console.log("Settings", e);
  broadcast(e);
});

// Also broadcast all messages on the `discord` channel. These are published on
// a separate channel to avoid clobbering the rewind window for sync.
const discord = client.channels.get("discord");
discord.subscribe("m", (e: any) => {
  e.event = e.name;
  broadcast(e);
});

// Notify clients of connection state changes.
client.connection.on("connected", () => {
  console.log("Connected");
  state = "connected";
  broadcast({ event: "_", state });
});
client.connection.on("disconnected", () => {
  console.log("Disconnected");
  state = "disconnected";
  broadcast({ event: "_", state });
});

// After about two minutes offline, uninterrupted in-order message delivery is
// no longer possible. Terminate the worker and instruct all clients to reload
// the page.
client.connection.on("suspended", () => {
  console.log("Terminating...");
  state = "broken";
  broadcast({ event: "_", state });
  client.close();
  close();
});

console.log("Ably worker initialized");
