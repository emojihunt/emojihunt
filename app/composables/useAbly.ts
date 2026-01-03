import AblySharedWorker from "~/ablyWorker?sharedworker";
import AblyDedicatedWorker from "~/ablyWorker?worker";
import type { AblyWorkerMessage, DiscordMessage, SettingsMessage, SyncMessage } from "~/utils/types";

export default function (
  sync: (m: SyncMessage) => void,
  settings: (m: SettingsMessage) => void,
  discord: (m: DiscordMessage) => void): Ref<boolean> {
  const connected = ref<boolean>(false);
  let poisoned = false;

  const onMessage = (e: MessageEvent<AblyWorkerMessage>) => {
    if (e.data.event === "sync") {
      sync(e.data.data);
    } else if (e.data.event === "settings") {
      settings(e.data.data);
    } else if (e.data.event === "m") {
      discord(e.data.data);
    } else if (e.data.event === "client") {
      if (e.data.state === "connected") {
        if (poisoned) window.location.reload();
        else connected.value = true;
      } else if (e.data.state === "disconnected") {
        connected.value = false;
      } else {
        console.warn("Connection lost. Will reload page when next online...");
        poisoned = true;
      }
    }
  };
  const onError = (e: Event) => console.warn("Worker Error:", e);
  onMounted(() => {
    // SharedWorker is only available on the client...
    if (typeof SharedWorker === "undefined") {
      // ...and isn't available in Chrome for Android.
      const worker = new AblyDedicatedWorker({ name: "🌊🎨🎡 ·⚡" });
      worker.addEventListener("message", onMessage);
      worker.addEventListener("error", onError);
      worker.postMessage("start");
    } else {
      const worker = new AblySharedWorker({ name: "🌊🎨🎡 ·⚡" });
      worker.port.addEventListener("message", onMessage);
      worker.port.addEventListener("error", onError);
      worker.port.start();
    }
  });
  return connected;
}
