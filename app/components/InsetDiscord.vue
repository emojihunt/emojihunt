<script setup lang="ts">
const { id } = defineProps<{ id: number; }>();
const emit = defineEmits<{ (e: "open"): void; }>();

const { discordCallback, puzzles, settings } = usePuzzles();
const [discordBase, discordTarget] = useDiscordBase();
const puzzle = computed(() => puzzles.get(id)!);

const channel = ref("");
const messages = useDiscordChannel(channel, discordCallback);
watchEffect(() => channel.value = puzzle.value.discord_channel || "");

const discordURL = computed(() =>
  `${discordBase}/channels/${settings.discordGuild}/${puzzle.value.discord_channel}`
);

const open = ref(false);
const [mute, setMute] = useMute();

const input = useTemplateRef("input");
const draft = ref("");

const filtered = computed(() =>
  [...messages.values()].sort((a, b) => a.t - b.t)
);
watch(() => filtered.value?.length, () => {
  if (!filtered.value?.length) {
    open.value = false;
  } else if (!mute.value) {
    open.value = true;
    emit("open");
    nextTick(() => input.value?.focus());
  }
});

const escape = (e: KeyboardEvent) => {
  if (e.key === "Escape") open.value = false;
};
const send = (e: KeyboardEvent) => {
  if (e.key === "Enter") {
    if (!draft.value) return;
    fetch(`/api/puzzles/${id}/messages`, {
      method: "POST",
      headers: {
        "Content-Type": "application/x-www-form-urlencoded",
      },
      body: (new URLSearchParams({ msg: draft.value })).toString(),
    });
    draft.value = "";
  }
};

defineExpose({
  toggle(): void {
    open.value = !open.value;
    if (open.value) {
      emit("open");
      nextTick(() => input.value?.focus());
    };
  },
});
</script>

<template>
  <div class="discord" v-if="open" @keydown="escape">
    <span class="links">
      <ETooltip text="Close" placement="top" :offset-distance="4">
        <button @click="() => (open = false)">
          🔻
        </button>
      </ETooltip>
      &bull;
      <ETooltip :text="mute ? 'Unmute' : 'Mute'" placement="top" :offset-distance="4">
        <button @click="() => setMute(!mute)">
          {{ mute ? "🔕" : "🔔" }}
        </button>
      </ETooltip>
      &bull;
      <ETooltip text="Open in Discord" placement="top" :offset-distance="4">
        <NuxtLink :to="discordURL" :target="discordTarget" :ok="!!puzzle.discord_channel">
          👉
        </NuxtLink>
      </ETooltip>
    </span>
    <p v-for="message of filtered">
      <b>{{ message.u }} &centerdot;</b> {{ message.msg }}
    </p>
    <input ref="input" type=text v-model="draft" placeholder="Reply..." @keydown="send" />
  </div>
</template>

<style scoped>
.discord {
  padding: 9px 9px 0 0;

  display: flex;
  flex-direction: column;
  gap: 4px;

  font-size: 13px;
  color: white;
}

p,
input {
  padding: 8px 12px;

  max-height: 150px;

  border: 1px solid #e1e3e1;
  border-radius: 6px;
  background-color: rgb(49 51 48 / 80%);

  outline-offset: -2px;
  user-select: text;
}

p:hover {
  max-height: unset;
}

.links {
  font-size: 12px;
  color: #313338;
  text-align: right;
}
</style>
