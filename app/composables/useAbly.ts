import AblyWorker from "~/ablyWorker?sharedworker";
import type { AblyWorkerMessage } from "~/utils/types";

export default function (): Ref<boolean> {
  const store = usePuzzles();
  const connected = ref<boolean>(false);
  let poisoned = false;
  onMounted(() => {
    // SharedWorker is only available on the client.
    const worker = new AblyWorker();
    worker.port.addEventListener("message", (e: MessageEvent<AblyWorkerMessage>) => {
      if (e.data.name === "sync") {
        store.handleDelta(e.data.data);
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
