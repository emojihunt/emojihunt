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
  useHead({
    title: puzzle.name,
    link: [{ rel: "icon", href: `https://emojicdn.elk.sh/${round.emoji}?style=google` }],
  });
  const spreadsheetURL = puzzle.spreadsheet_id ?
    `https://docs.google.com/spreadsheets/d/${puzzle.spreadsheet_id}` : "";
  const discordURL =
    `${discordBase}/channels/${store.discordGuild}/${puzzle.discord_channel}`;
  return { puzzle, round, discordURL, spreadsheetURL };
});
</script>

<template>
  <iframe :src="data?.spreadsheetURL || ''">
    Spreadsheet failed to load.
    <NuxtLink :to="data?.spreadsheetURL">Go to Google Sheets</NuxtLink>.
  </iframe>
  <nav>
    <section>
      <ETooltip text="Click to set status to ✍️ Working" placement="top"
        :offset-distance="4" v-if="data.puzzle.status === Status.NotStarted">
        <button
          @click="() => store.updatePuzzleOptimistic(data.puzzle.id, { status: Status.Working })">
          ‼️
        </button>
      </ETooltip>
      <ETooltip :text="`Status: ${StatusLabel(data.puzzle.status)}`" placement="top"
        :offset-distance="4" v-else>
        {{ StatusEmoji(data.puzzle.status) }}
      </ETooltip>
    </section>
    <section>
      <ETooltip text="Puzzle Page" placement="top" :offset-distance="4">
        <NuxtLink :to="data.puzzle.puzzle_url" target="TODO"
          :ok="!!data.puzzle.puzzle_url">
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

<style>
html,
body,
#__nuxt,
#__nuxt>div {
  width: 100%;
  height: 100%;
}
</style>

<style scoped>
iframe {
  width: 100%;
  height: 100%;
}

nav {
  position: fixed;
  bottom: 0;
  right: 0;

  height: 37px;
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
  border-left: 1px solid #e1e3e1;
  background-color: rgb(249 251 253 / 50%);

  user-select: none;
}

.logo {
  letter-spacing: 0.166em;
  opacity: 70%;
  cursor: default;
}
</style>
