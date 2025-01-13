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
  discordCallback, settings, puzzles, rounds,
  voiceRooms, updatePuzzleOptimistic,
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
  puzzle.value.voice_room && voiceRooms.get(puzzle.value.voice_room)
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
</script>

<template>
  <main :class="split">
    <iframe :src="spreadsheetURL || ''"></iframe>
    <iframe :src="puzzleURL" class="puzzle"></iframe>
  </main>
  <nav :class="split">
    <section>
      <ETooltip text="Click to set status to ✍️ Working" placement="top"
        :offset-distance="4"
        v-if="puzzle.status === Status.NotStarted || puzzle.status === Status.Abandoned">
        <button
          @click="() => updatePuzzleOptimistic(puzzle.id, { status: Status.Working })">
          {{ StatusEmoji(puzzle.status) || "‼️" }}
        </button>
      </ETooltip>
      <ETooltip :text="`Status: ${StatusLabel(puzzle.status)}`" placement="top"
        :offset-distance="4" v-else>
        {{ StatusEmoji(puzzle.status) }}
      </ETooltip>
      <ETooltip :text="`Voice Room: ${voiceRoom.name}`" placement="top"
        :offset-distance="4" v-if="voiceRoom">
        {{ voiceRoom.emoji }}
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
        <NuxtLink :to="discordURL" :target="discordTarget" :ok="!!puzzle.discord_channel">
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

nav {
  position: fixed;
  bottom: 0;
  right: 0;

  height: 36px;
  max-width: 75%;
  padding: 1px 0.5em 0;

  display: flex;
  align-items: center;
}

section {
  display: flex;
  align-items: center;

  margin: 0 0.6em;
  gap: 0.33em;
}

/* Theming */
main {
  user-select: none;
}

nav {
  font-size: 15px;
  border-left: 1px solid #e1e3e1;
  user-select: none;
}

nav.split {
  background-color: rgb(249 251 253 / 75%);
  border-top: 1px solid #e1e3e1;
  border-top-left-radius: 8px;
}

.logo {
  letter-spacing: 0.166em;
  opacity: 70%;
  cursor: default;
}
</style>
