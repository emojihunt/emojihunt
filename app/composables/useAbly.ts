import AblyWorker from "~/ablyWorker?sharedworker";

export default function (): void {
  const store = usePuzzles();
  onMounted(() => {
    // SharedWorker is only available on the client.
    const worker = new AblyWorker();
    worker.port.addEventListener("message", (e) => {
      if (e.data.name === "sync") {
        store.handleUpdate(e.data.data);
      }
    });
    worker.addEventListener(
      "error", (e) => console.warn("Worker Error:", e)
    );
    worker.port.start();
  });
}
