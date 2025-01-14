const key = "muted";

export default function (): [Ref<boolean>, (v: boolean) => void] {
  const mute = ref(false);
  onMounted(() => mute.value = (localStorage.getItem(key) === "1"));

  const channel = (new BroadcastChannel(key));
  channel.addEventListener("message", (e) => (mute.value = !!e.data));

  const setMute = (v: boolean) => {
    mute.value = v;
    localStorage.setItem(key, v ? "1" : "0");
    channel.postMessage(v);
  };
  return [mute, setMute];
}
