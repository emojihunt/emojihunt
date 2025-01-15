export default function (
  channel: Ref<string>,
  callback: Ref<((m: DiscordMessage) => void) | undefined>,
): Map<string, DiscordMessage> {

  if (import.meta.server) return new Map();

  const messages = reactive(new Map<string, DiscordMessage>());
  callback.value = (m: DiscordMessage) => {
    const prev = messages.get(m.id);
    if (prev) {
      // @ts-ignore
      if (m.t === 0) delete m.t; // serialization bug

      if (m.msg) Object.assign(prev, m); // update
      else messages.delete(m.id); // delete
    } else if (m.ch === channel.value && m.u) {
      // (ignore updates if we don't have the original)
      messages.set(m.id, m); // create
    }
  };
  return messages;
};
