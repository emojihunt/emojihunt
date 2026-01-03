import AblySharedWorker from "~/ablyWorker?sharedworker";
import AblyDedicatedWorker from "~/ablyWorker?worker";
import LiveSharedWorker from "~/liveWorker?sharedworker";
import LiveDedicatedWorker from "~/liveWorker?worker";
import type { AblyWorkerMessage, DiscordMessage, SettingsMessage, SyncMessage } from "~/utils/types";

export default function (
  sync: (m: SyncMessage) => void,
  settings: (m: SettingsMessage) => void,
  discord: (m: DiscordMessage) => void,
): Ref<boolean> {
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
      } else if (e.data.state === "dead") {
        console.warn("Live server got too far ahead. Reloading page...");
        window.location.reload();
      } else { // "broken" (Ably only)
        console.warn("Connection lost. Will reload page when next online...");
        poisoned = true;
      }
    }
  };
  const onError = (e: Event) => console.warn("Worker Error:", e);
  onMounted(() => {
    const { newSyncBackend } = useAppConfig();
    // SharedWorker is only available on the client...
    if (typeof SharedWorker === "undefined") {
      // ...and isn't available in Chrome for Android.
      const worker = newSyncBackend
        ? new LiveDedicatedWorker({ name: "🌊🎨🎡 ·⚡⚡" })
        : new AblyDedicatedWorker({ name: "🌊🎨🎡 ·⚡" });
      worker.addEventListener("message", onMessage);
      worker.addEventListener("error", onError);
      worker.postMessage("start");
    } else {
      const id = crypto.randomUUID();
      navigator.locks.request(id, () => new Promise(() => {
        const worker = newSyncBackend
          ? new LiveSharedWorker({ name: "🌊🎨🎡 ·⚡⚡" })
          : new AblySharedWorker({ name: "🌊🎨🎡 ·⚡" });
        worker.port.addEventListener("message", onMessage);
        worker.port.addEventListener("error", onError);
        worker.port.start();
        worker.port.postMessage(id);
      }));
    }
  });
  return connected;
}
