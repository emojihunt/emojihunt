import AblyWorker from "~/ablyWorker?sharedworker";
import type { AblyWorkerMessage } from "~/utils/types";

export default function (): Ref<boolean> {
  const store = usePuzzles();
  const connected = ref<boolean>(false);
  onMounted(() => {
    // SharedWorker is only available on the client.
    const worker = new AblyWorker();
    worker.port.addEventListener("message", (e: MessageEvent<AblyWorkerMessage>) => {
      if (e.data.name === "sync") {
        store.handleDelta(e.data.data);
      } else if (e.data.name === "client") {
        if (e.data.state === "connected") {
          connected.value = true;
        } else if (e.data.state === "disconnected") {
          connected.value = false;
        } else {
          if (window.navigator.onLine) {
            console.warn("Connection lost. Reloading page...");
            window.location.reload();
          } else {
            console.warn("Connection lost. Will reload page when next online...");
            window.addEventListener("online", () => window.location.reload());
          }
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
