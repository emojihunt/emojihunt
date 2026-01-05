import AblySharedWorker from "~/ablyWorker?sharedworker";
import AblyDedicatedWorker from "~/ablyWorker?worker";
import LiveSharedWorker from "~/liveWorker?sharedworker";
import LiveDedicatedWorker from "~/liveWorker?worker";
import type { AblyWorkerMessage, DiscordMessage, SettingsMessage, SyncMessage } from "~/utils/types";

const INACTIVITY_TIMEOUT = 15 * 60 * 1000; // 15 minutes
const ACTIVITY_CHECK_INTERVAL = 60 * 1000; // 60 seconds

export default function (
  puzzle: number | undefined,
  sync: (m: SyncMessage) => void,
  settings: (m: SettingsMessage) => void,
  discord: (m: DiscordMessage) => void,
): [Ref<boolean>, Ref<boolean>] {
  const connected = ref<boolean>(false);
  const active = ref<boolean>(false);
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
    const id = crypto.randomUUID();
    let port: Worker | MessagePort;
    const { newSyncBackend } = useAppConfig();

    // SharedWorker is only available on the client...
    if (typeof SharedWorker === "undefined") {
      // ...and isn't available in Chrome for Android.
      const worker = newSyncBackend
        ? new LiveDedicatedWorker({ name: "🌊🎨🎡 ·⚡⚡" })
        : new AblyDedicatedWorker({ name: "🌊🎨🎡 ·⚡" });
      worker.addEventListener("message", onMessage);
      worker.addEventListener("error", onError);
      worker.postMessage({ event: "start", id });
      port = worker;
    } else {
      const worker = newSyncBackend
        ? new LiveSharedWorker({ name: "🌊🎨🎡 ·⚡⚡" })
        : new AblySharedWorker({ name: "🌊🎨🎡 ·⚡" });
      worker.port.addEventListener("message", onMessage);
      worker.port.addEventListener("error", onError);
      worker.port.start();
      port = worker.port;
      navigator.locks.request(id, () => new Promise(() =>
        worker.port.postMessage({ event: "start", id })
      ));
    }

    // Activity tracking. Times are stored in milliseconds since page load
    // (digits after the decimal point are microseconds). Get current value with
    // `performance.now()`.
    if (puzzle) {
      let lastActivity = performance.now();
      const checkActivity = () => {
        const newActive = lastActivity + INACTIVITY_TIMEOUT > performance.now();
        if (active.value !== newActive) {
          active.value = newActive;
          port.postMessage({ event: "activity", id, puzzle, active: newActive });
        }
      };
      checkActivity(); // will set to true
      setInterval(checkActivity, ACTIVITY_CHECK_INTERVAL);

      const onActivity = (ts: number) => {
        lastActivity = ts;
        if (!active.value) checkActivity();
      };
      window.addEventListener("blur", (e) => setTimeout(() =>
        (document.activeElement?.nodeName === "IFRAME") && onActivity(e.timeStamp), 250)
      );
      window.addEventListener("mousemove", (e) => onActivity(e.timeStamp));
      window.addEventListener("click", (e) => onActivity(e.timeStamp));
      window.addEventListener("keydown", (e) => onActivity(e.timeStamp));
    }
  });
  return [connected, active];
}
