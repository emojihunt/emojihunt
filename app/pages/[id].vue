<script setup lang="ts">
definePageMeta({
  validate: (route) => {
    const id = parseInt(route.params.id as string);
    return (id.toString() === route.params.id);
  },
});

useHead({ htmlAttrs: { lang: "en" } });
const route = useRoute();
const {
  discordCallback, settings, puzzles, rounds, voiceRooms
} = await initializePuzzles();
const channel = ref("");
const messages = useDiscordChannel(channel, discordCallback);
const [discordBase, discordTarget] = useDiscordBase();

const puzzle = computed(() => {
  const id = parseInt(route.params.id as string);
  const puzzle = puzzles.get(id);
  if (!puzzle) {
    throw createError({
      fatal: true,
      message: `Puzzle #${id} could not be found`,
      statusCode: 404,
    });
  };
  if (!puzzle.spreadsheet_id) {
    throw createError({
      fatal: true,
      message: `Puzzle #${id} does not have a spreadsheet`,
      statusCode: 404,
    });
  }
  return puzzle;
});
const round = computed(() =>
  puzzle.value && rounds.get(puzzle.value.round)
);


watchEffect(() => useHead({
  title: puzzle.value.name,
  link: [{ rel: "icon", key: "icon", href: `https://emojicdn.elk.sh/${round.value?.emoji}?style=google` }],
}));
watchEffect(() => channel.value = puzzle.value?.discord_channel);
const voiceRoom = computed(() =>
  puzzle.value.voice_room ? voiceRooms.get(puzzle.value.voice_room) : undefined
);
const spreadsheetURL = computed(() =>
  puzzle.value.spreadsheet_id
    ? `https://docs.google.com/spreadsheets/d/${puzzle.value.spreadsheet_id}`
    : ""
);
const discordURL = computed(() =>
  `${discordBase}/channels/${settings.discordGuild}/${puzzle.value.discord_channel}`
);

onBeforeMount(() => document.body.classList.add("fullscreen"));

const puzzleURL = ref("");
const split = ref("");
const togglePuzzle = (e: MouseEvent) => {
  if (!puzzle.value?.puzzle_url) return;
  if (e.metaKey || e.ctrlKey) return; // open in new tab

  e.preventDefault();
  if (!puzzleURL.value) {
    // Lazy-load the puzzle frame
    puzzleURL.value = puzzle.value.puzzle_url;
  }
  split.value = split.value ? "" : "split";
};

const open = ref<"status" | "voice">();
</script>

<template>
  <main :class="split">
    <iframe :src="spreadsheetURL || ''"></iframe>
    <iframe :src="puzzleURL" class="puzzle"></iframe>
  </main>
  <div :class="['inset', split]">
    <nav @keydown="(e) => e.key === 'Escape' && (open = undefined)">
      <section>
        <ETooltip :text="`Status: ${StatusLabel(puzzle.status)}`" placement="top"
          :offset-distance="4">
          <button @click="() => (open = (open === 'status') ? undefined : 'status')">
            {{ StatusEmoji(puzzle.status) || " ‼️" }} </button>
        </ETooltip>
        <ETooltip :text="voiceRoom ? `Voice Room: ${voiceRoom.name}` : 'No Voice Room'"
          placement="top" :offset-distance="4">
          <button :class="!voiceRoom && 'unset'"
            @click="() => (open = (open === 'voice') ? undefined : 'voice')">
            {{ voiceRoom?.emoji || "📻" }}
          </button>
        </ETooltip>
      </section>
      <section>
        <ETooltip text="Puzzle Page" placement="top" :offset-distance="4">
          <NuxtLink :to="puzzle.puzzle_url" :ok="!!puzzle.puzzle_url" target="_blank"
            @click="togglePuzzle">
            🌎
          </NuxtLink>
        </ETooltip>
        <ETooltip text="Discord Channel" placement="top" :offset-distance="4">
          <NuxtLink :to="discordURL" :target="discordTarget"
            :ok="!!puzzle.discord_channel">
            👾
          </NuxtLink>
        </ETooltip>
      </section>
      <section>
        <ETooltip :text="`Round: ${round?.name}`" placement="top" :offset-distance="4">
          {{ round?.emoji }}
        </ETooltip>
      </section>
      <section>
        <NuxtLink to="/" :external="true" class="logo">
          <span>🌊🎨🎡</span>
        </NuxtLink>
      </section>
    </nav>
    <InsetStatus v-if="open === 'status'" :id="puzzle.id"
      @close="() => (open = undefined)" />
    <InsetVoice v-if="open === 'voice'" :id="puzzle.id"
      @close="() => (open = undefined)" />
  </div>
</template>

<style scoped>
main {
  width: 100%;
  height: 100%;

  display: flex;
}

iframe {
  width: 100%;
  height: 100%;
}

.puzzle {
  display: none;
}

.split .puzzle {
  display: unset;
}

.inset {
  position: fixed;
  bottom: 0;
  right: 0;

  max-width: 75%;

  display: flex;
  flex-direction: column-reverse;
  gap: 8px;
}

nav {
  height: 36px;
  padding: 1px 0.5em 0;
  flex-shrink: 0;

  display: flex;
  align-items: center;
}

section {
  display: flex;
  align-items: center;

  margin: 0 0.6em;
  gap: 8px;
}

/* Theming */
main,
.inset {
  user-select: none;
}

.inset {
  font-size: 15px;
}

nav {
  border-left: 1px solid #e1e3e1;
}

.split nav {
  background-color: rgb(249 251 253 / 75%);
  border-top: 1px solid #e1e3e1;
  border-top-left-radius: 6px;
}

.logo {
  letter-spacing: 0.166em;
  opacity: 70%;
  cursor: default;
}

button.unset {
  filter: grayscale(100%) opacity(60%);
}
</style>
