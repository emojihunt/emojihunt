<script setup lang="ts">
definePageMeta({
  validate: (route) => {
    const id = parseInt(route.params.id as string);
    return (id.toString() === route.params.id);
  },
});

useHead({
  htmlAttrs: { lang: "en" },
  meta: [
    { name: "viewport", content: "width=device-width, initial-scale=1, viewport-fit=cover" },
    { name: "theme-color", content: "oklch(30% 0 0deg)" },
  ],
});
const route = useRoute();
const { puzzles, rounds, voiceRooms, settings } = await initializePuzzles();

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
const voiceRoom = computed(() =>
  puzzle.value.voice_room ? voiceRooms.get(puzzle.value.voice_room) : undefined
);
const spreadsheetURL = computed(() =>
  puzzle.value.spreadsheet_id
    ? `https://docs.google.com/spreadsheets/d/${puzzle.value.spreadsheet_id}`
    : ""
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

const discord = useTemplateRef("discord");
const [discordBase, discordTarget] = useDiscordBase();
const discordURL = computed(() => puzzle.value.discord_channel ?
  `${discordBase}/channels/${settings.discordGuild}/${puzzle.value.discord_channel}` : "");
const toggleDiscord = (e: MouseEvent) => {
  if (!discordURL.value) return;
  if (e.metaKey || e.ctrlKey) return; // open in new tab
  e.preventDefault();
  discord.value?.toggle();
};

const insets = useTemplateRef("insets");
const open = ref<"status" | "voice">();
const toggle = (kind: "status" | "voice") => {
  if (open.value === kind) {
    open.value = undefined;
  } else {
    open.value = kind;
    nextTick(() => insets.value?.scrollTo({ top: 99999 }));
  }
};

// On mobile, Google Sheets opens in "htmlview" mode by default. Show a banner
// to open the native app, since that's much better.
const applink = ref("");
onMounted(() => {
  if (/Android/i.test(navigator.userAgent)) {
    applink.value = spreadsheetURL.value;
  } else if (/iPhone|iPad|iPod/i.test(navigator.userAgent)) {
    applink.value = `googlesheets://sheets.google.com/spreadsheets/d/${puzzle.value.spreadsheet_id}`;
  }
});
</script>

<template>
  <main :class="split">
    <div>
      <NuxtLink :to="applink" class="applink" v-if="!!applink">
        Open in Google Sheets &nbsp;
      </NuxtLink>
      <iframe :src="spreadsheetURL"></iframe>
    </div>
    <iframe :src="puzzleURL" class="puzzle"></iframe>
  </main>
  <div :class="['overlay', split]">
    <nav @keydown="(e) => e.key === 'Escape' && (open = undefined)">
      <section>
        <ETooltip :text="`Status: ${StatusLabel(puzzle.status)}`" placement="top"
          :offset-distance="4">
          <button @click="() => toggle('status')">
            {{ StatusEmoji(puzzle.status) || " ‼️" }} </button>
        </ETooltip>
        <ETooltip :text="voiceRoom ? `Voice Room: ${voiceRoom.name}` : puzzle.location ?
          `In-person: ${puzzle.location}` : 'No Voice Room'" placement="top"
          :offset-distance="4">
          <button :class="!voiceRoom && !puzzle.location && 'unset'"
            @click="() => toggle('voice')">
            {{ voiceRoom?.emoji || (puzzle.location ? "📍" : "📻") }}
          </button>
        </ETooltip>
      </section>
      <section>
        <ETooltip text="Puzzle Page" placement="top" :offset-distance="4">
          <NuxtLink :to="puzzle.puzzle_url" target="_blank" @click="togglePuzzle">
            🌎
          </NuxtLink>
        </ETooltip>
        <ETooltip text="Discord Channel" placement="top" :offset-distance="4">
          <NuxtLink :to="discordURL" :target="discordTarget" :ok="!!discordURL"
            @click="toggleDiscord">
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
    <span ref="insets" class="insets">
      <InsetStatus v-if="open === 'status'" :id="puzzle.id"
        @close="() => (open = undefined)" />
      <InsetVoice v-if="open === 'voice'" :id="puzzle.id"
        @close="() => (open = undefined)" />
      <InsetDiscord ref="discord" :id="puzzle.id"
        @open="() => nextTick(() => insets?.scrollTo({ top: 99999 }))" />
    </span>
  </div>
</template>

<style scoped>
main {
  width: 100%;
  height: 100dvh;

  display: flex;
}

main>div,
iframe {
  display: flex;
  flex-direction: column;
  flex-grow: 1;
}

.puzzle {
  display: none;
}

.split .puzzle {
  display: unset;
}

.overlay {
  position: fixed;
  bottom: 0;
  right: 0;

  width: 260px;
  max-height: 100%;
}

.overlay,
.insets {
  display: flex;
  flex-direction: column-reverse;
  gap: 8px;
}

nav {
  height: 36px;
  padding: 1px 0 0;
  flex-shrink: 0;

  display: flex;
  align-items: center;
  justify-content: center;
}

section {
  display: flex;
  align-items: center;

  margin: 0 9px;
  gap: 8px;
}

/* Theming */
main,
.overlay {
  font-size: 15px;
  user-select: none;
}

nav {
  border-left: 1px solid #e1e3e1;
}

.insets {
  overflow-y: scroll;
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

a[ok="false"] {
  filter: grayscale(100%) opacity(60%);
  pointer-events: none;
}

.applink {
  padding: 1rem;
  padding-right: calc(1rem + env(safe-area-inset-right));
  border-radius: 0;

  font-weight: 600;
  text-align: right;

  color: white;
  background-color: oklch(52.7% 0.141 148.39);
}
</style>
