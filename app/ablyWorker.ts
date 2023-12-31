import Ably from 'ably/build/ably-webworker.min';
import type { AblyWorkerMessage } from './utils/types';

// Keep a list of clients as they connect. There's no easy way to implement
// garbage collection, unfortunately...
const ports: Array<MessagePort> = [];
const broadcast = (m: AblyWorkerMessage) => ports.forEach(p => p.postMessage(m));
self.addEventListener("connect", (e: any) => {
  console.log("Client connected");
  ports.push(...e.ports);
});

const client = new Ably.Realtime.Promise({
  authCallback: async (_, callback) => {
    console.log("Fetching Ably token...");
    const r = await fetch("/api/ably", { method: "POST" });
    if (r.status != 200) {
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
const channel = client.channels.get("huntbot", { params: { rewind: "2m" } });

// Broadcast all messages on the `huntbot` channel to all clients.
channel.subscribe("sync", (e) => (console.log("Sync", e), broadcast(e as any)));

// Notify clients of connection state changes.
client.connection.on("connected", () => (
  console.log("Connected"), broadcast({ name: "client", state: "connected" })
));
client.connection.on("disconnected", () => (
  console.log("Disconnected"), broadcast({ name: "client", state: "disconnected" })
));

// After about two minutes offline, uninterrupted in-order message delivery is
// no longer possible. Terminate the worker and instruct all clients to reload
// the page.
client.connection.on("suspended", () => {
  console.log("Terminating...");
  broadcast({ name: "client", state: "broken" });
  client.close();
  close();
});

console.log("Ably worker initialized");
