<script setup lang="ts">
useHead({ htmlAttrs: { lang: "en" } });
const route = useRoute();
const store = usePuzzles();
await store.refresh();
const connected = useAbly();
const [discordBase, discordTarget] = useDiscordBase();

const data = computed(() => {
  const id = parseInt(route.params.id as string);
  if (id.toString() !== route.params.id) {
    throw createError({
      fatal: true,
      message: `Page not found: ${route.fullPath}`,
      statusCode: 404,
    });
  }

  const puzzle = store.puzzles.get(id);
  const round = store.rounds.find((r) => r.id === puzzle?.round);
  if (!puzzle || !round) {
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


  const voiceRoom = puzzle.voice_room && store.voiceRooms.get(puzzle.voice_room);
  const spreadsheetURL = puzzle.spreadsheet_id ?
    `https://docs.google.com/spreadsheets/d/${puzzle.spreadsheet_id}` : "";
  const discordURL =
    `${discordBase}/channels/${store.discordGuild}/${puzzle.discord_channel}`;
  return { puzzle, round, voiceRoom, discordURL, spreadsheetURL };
});

const setHead = () => useHead({
  title: data.value.puzzle.name,
  link: [{ rel: "icon", key: "icon", href: `https://emojicdn.elk.sh/${data.value.round.emoji}?style=google` }],
});
onMounted(setHead);
watch(() => [data.value.puzzle.name, data.value.round], setHead);

onBeforeMount(() => document.body.classList.add("fullscreen"));

const puzzleURL = ref("");
const puzzleDisplay = ref("none");
const togglePuzzle = (e: MouseEvent) => {
  if (!data.value?.puzzle.puzzle_url) return;
  if (e.metaKey || e.ctrlKey) return; // open in new tab

  e.preventDefault();
  if (!puzzleURL.value) {
    // Lazy-load the puzzle frame
    puzzleURL.value = data.value.puzzle.puzzle_url;
  }
  if (puzzleDisplay.value === "unset") {
    puzzleDisplay.value = "none";
  } else {
    puzzleDisplay.value = "unset";
  }
};
</script>

<template>
  <main>
    <iframe :src="data?.spreadsheetURL || ''"></iframe>
    <iframe :src="puzzleURL" class="puzzle"></iframe>
  </main>
  <nav>
    <section>
      <ETooltip text="Click to set status to ✍️ Working" placement="top"
        :offset-distance="4"
        v-if="data.puzzle.status === Status.NotStarted || data.puzzle.status === Status.Abandoned">
        <button
          @click="() => store.updatePuzzleOptimistic(data.puzzle.id, { status: Status.Working })">
          {{ StatusEmoji(data.puzzle.status) || "‼️" }}
        </button>
      </ETooltip>
      <ETooltip :text="`Status: ${StatusLabel(data.puzzle.status)}`" placement="top"
        :offset-distance="4" v-else>
        {{ StatusEmoji(data.puzzle.status) }}
      </ETooltip>
      <ETooltip :text="`Voice Room: ${data.voiceRoom.name}`" placement="top"
        :offset-distance="4" v-if="data.voiceRoom">
        {{ data.voiceRoom.emoji }}
      </ETooltip>
    </section>
    <section>
      <ETooltip text="Puzzle Page" placement="top" :offset-distance="4">
        <NuxtLink :to="data.puzzle.puzzle_url" :ok="!!data.puzzle.puzzle_url"
          target="_blank" @click="togglePuzzle">
          🌎
        </NuxtLink>
      </ETooltip>
      <ETooltip text="Discord Channel" placement="top" :offset-distance="4">
        <NuxtLink :to="data.discordURL" :target="discordTarget"
          :ok="!!data.puzzle.discord_channel">
          👾
        </NuxtLink>
      </ETooltip>
    </section>
    <section>
      <ETooltip :text="`Round: ${data.round.name}`" placement="top" :offset-distance="4">
        {{ data.round.emoji }}
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
  display: v-bind(puzzleDisplay);
}

nav {
  position: fixed;
  bottom: 0;
  right: 0;

  height: 37.5px;
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
nav {
  font-size: 15px;
  background-color: rgb(249 251 253 / 66%);

  border-left: 1px solid #e1e3e1;
  border-top: 1px solid #e1e3e1;
  border-top-left-radius: 6px;

  user-select: none;
}

.logo {
  letter-spacing: 0.166em;
  opacity: 70%;
  cursor: default;
}
</style>
