import Ably from 'ably/build/ably-webworker.min';

const ports: Array<MessagePort> = [];
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
client.channels.get("[?rewind=2m]huntbot").subscribe("sync", (e) => {
  console.log("Sync", e);
  ports.forEach(p => p.postMessage(e));
});

console.log("Ably worker initialized");
