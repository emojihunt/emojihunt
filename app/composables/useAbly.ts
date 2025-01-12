import AblyWorker from "~/ablyWorker?sharedworker";
import type { AblyWorkerMessage, SyncMessage } from "~/utils/types";

export default function (handleDelta: (m: SyncMessage) => void): Ref<boolean> {
  const connected = ref<boolean>(false);
  let poisoned = false;
  onMounted(() => {
    // SharedWorker is only available on the client.
    const worker = new AblyWorker({ name: "🌊🎨🎡 ·⚡" });
    worker.port.addEventListener("message", (e: MessageEvent<AblyWorkerMessage>) => {
      if (e.data.name === "sync") {
        handleDelta(e.data.data);
      } else if (e.data.name === "client") {
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
    });
    worker.addEventListener(
      "error", (e) => console.warn("Worker Error:", e)
    );
    worker.port.start();
  });
  return connected;
}
