const key = "muted";

export default function (): [Ref<boolean>, (v: boolean) => void] {
  let channel: BroadcastChannel | undefined;

  const mute = ref(false);
  const setMute = (v: boolean) => {
    mute.value = v;
    localStorage.setItem(key, v ? "1" : "0");
    channel?.postMessage(v);
  };

  onMounted(() => {
    mute.value = (localStorage.getItem(key) === "1");

    channel = new BroadcastChannel(key);
    channel.addEventListener("message", (e) => (mute.value = !!e.data));
  });
  return [mute, setMute];
}
