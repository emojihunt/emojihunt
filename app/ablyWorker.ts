import Ably from 'ably';
import type { AblyWorkerMessage, ConnectionState } from './utils/types';

// Keep a list of clients as they connect. There's no easy way to implement
// garbage collection, unfortunately...
const ports: Array<MessagePort> = [];
const broadcast = (m: AblyWorkerMessage) => ports.forEach(p => p.postMessage(m));
self.addEventListener("connect", (e: any) => {
  console.log("Client connected");
  for (const port of e.ports) {
    ports.push(port);
    port.postMessage({ name: "client", state });
    rewind.forEach(r => port.postMessage(r));
  }
});

const client = new Ably.Realtime({
  authCallback: async (_, callback) => {
    console.log("Fetching Ably token...");
    const r = await fetch("/api/ably", { method: "POST" });
    if (r.status !== 200) {
      const msg = `HTTP ${r.status}: ${await r.text()}`;
      console.error(msg);
      callback(msg, null);
    } else {
      console.log("Received Ably token");
      callback(null, await r.json());
    }
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
const channel = client.channels.get("huntbot", { params: { rewind: "2m" } });
const rewind = new Array();
let state: ConnectionState = "disconnected";

// Broadcast all messages on the `huntbot` channel to all clients.
channel.subscribe("sync", (e) => {
  console.log("Sync", e);
  if (rewind.length >= 100) rewind.shift();
  rewind.push(e);
  broadcast(e as any);
});

// Notify clients of connection state changes.
client.connection.on("connected", () => {
  console.log("Connected");
  state = "connected";
  broadcast({ name: "client", state });
});
client.connection.on("disconnected", () => {
  console.log("Disconnected");
  state = "disconnected";
  broadcast({ name: "client", state });
});


// After about two minutes offline, uninterrupted in-order message delivery is
// no longer possible. Terminate the worker and instruct all clients to reload
// the page.
client.connection.on("suspended", () => {
  console.log("Terminating...");
  state = "broken";
  broadcast({ name: "client", state });
  client.close();
  close();
});

console.log("Ably worker initialized");
