import AblySharedWorker from "~/ablyWorker?sharedworker";
import AblyDedicatedWorker from "~/ablyWorker?worker";
import LiveSharedWorker from "~/liveWorker?sharedworker";
import LiveDedicatedWorker from "~/liveWorker?worker";
import type {
  AblyWorkerMessage, DiscordMessage, SettingsMessage, SyncMessage, UsersMessage,
} from "~/utils/types";

const INACTIVITY_TIMEOUT = 15 * 60 * 1000; // 15 minutes
const ACTIVITY_CHECK_INTERVAL = 60 * 1000; // 60 seconds

let previous = { set: false, puzzle: null as (number | null) };

export default function (
  puzzle: number | null,
  discord: (m: DiscordMessage) => void,
  presence: (m: PresenceMessage) => void,
  settings: (m: SettingsMessage) => void,
  sync: (m: SyncMessage) => void,
  users: (m: UsersMessage) => void,
): [Ref<boolean>, Ref<boolean>] {
  const connected = ref<boolean>(false);
  const active = ref<boolean>(true);
  let poisoned = false;

  const onMessage = (e: MessageEvent<AblyWorkerMessage>) => {
    switch (e.data.event) {
      case "_":
        switch (e.data.state) {
          case "connected":
            if (poisoned) window.location.reload();
            else connected.value = true;
            break;
          case "disconnected":
            connected.value = false;
            break;
          case "dead":
            console.warn("Live server got too far ahead. Reloading page...");
            window.location.reload();
            break;
          case "broken": // Ably only
            console.warn("Connection lost. Will reload page when next online...");
            poisoned = true;
            break;
          default:
            ((x: never) => console.warn("Unknown client state:", x))(e.data.state);
        }
        break;
      case "m":
        discord(e.data.data);
        break;
      case "presence":
        presence(e.data.data);
        break;
      case "settings":
        settings(e.data.data);
        break;
      case "sync":
        sync(e.data.data);
        break;
      case "users":
        users(e.data.data);
        break;
      default:
        ((x: never) => console.warn("Unknown event:", (x as any).event))(e.data);
    }
  };
  const onError = (e: Event) => console.warn("Worker Error:", e);
  onMounted(() => {
    if (previous.set && previous.puzzle !== puzzle) {
      throw new Error(`useAbly: cannot change puzzle: ${previous} -> ${puzzle}`);
    }
    previous = { set: true, puzzle };

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
      worker.postMessage({ event: "start", id, puzzle });
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
        worker.port.postMessage({ event: "start", id, puzzle })
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
      port.postMessage({ event: "activity", id, puzzle, active: active.value });
      setInterval(checkActivity, ACTIVITY_CHECK_INTERVAL);

      const onActivity = (ts: number, strong: boolean) => {
        // Mouse movement keeps us active, but stops counting once we're
        // inactive -- we need a click or keypress to wake.
        if (active.value) {
          lastActivity = ts;
        } else if (strong) {
          lastActivity = ts;
          // When waking up, notify pages immediately to make the UI responsive
          // (notifies the server, too).
          checkActivity();
        }
      };
      window.addEventListener("blur", (e) => setTimeout(() =>
        (document.activeElement?.nodeName === "IFRAME") && onActivity(e.timeStamp, false), 250)
      );
      window.addEventListener("mousemove", (e) => onActivity(e.timeStamp, false));
      window.addEventListener("click", (e) => onActivity(e.timeStamp, true));
      window.addEventListener("keydown", (e) => onActivity(e.timeStamp, true));
    }
  });
  return [connected, active];
};
