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
const muted = ref(false);
const toggleMute = () => {
  if (muted.value) {
    muted.value = false;
  } else {
    muted.value = true;
    open.value = false;
  }
};

const filtered = computed(() => {
  const now = Date.now();
  const cutoff = 30 * 60 * 1000; // 30 minutes
  return [...messages.values()]
    .filter((m) => (now - m.t) < cutoff)
    .sort((a, b) => a.t - b.t);
});
watch(() => filtered.value?.length, () => {
  if (!filtered.value?.length) {
    open.value = false;
  } else if (!muted.value) {
    open.value = true;
    emit("open");
  }
});

defineExpose({
  toggle(): void {
    open.value = !open.value;
    if (open.value) emit("open");
  },
});
</script>

<template>
  <div class="discord" v-if="open">
    <span class="links">
      <ETooltip text="Close" placement="top" :offset-distance="4">
        <button @click="() => (open = false)">
          🔻
        </button>
      </ETooltip>
      &bull;
      <ETooltip :text="muted ? 'Unmute' : 'Mute'" placement="top" :offset-distance="4">
        <button @click="toggleMute">
          {{ muted ? "🔕" : "🔔" }}
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
      <b>@{{ message.u }}:</b> {{ message.msg }}
    </p>
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

p {
  padding: 8px 12px;

  max-height: 150px;

  border: 1px solid #e1e3e1;
  border-radius: 6px;
  background-color: #313338;

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
