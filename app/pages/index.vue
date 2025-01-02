<script setup lang="ts">
useHead({
  htmlAttrs: { lang: "en" },
  title: "Puzzle Tracker",
});
const store = usePuzzles();
await store.refresh();
const connected = useAbly();

// Puzzle & Round Helpers
const timelineFromID = (id: number) => `--round-${id}`;
const nextTimelineFromID = (id: number): string | undefined =>
  store.rounds[id + 1] ? timelineFromID(id + 1) : undefined;

// It doesn't look great when the round headers stack up on top of one another.
// We want each round header to disappear when it's covered by the next. Use CSS
// scroll-linked animations if supported and fall back to IntersectionObserver
// if not.
const timelines = computed(() => store.rounds.map((_, i) => timelineFromID(i)));
const observer = ref<IntersectionObserver>();
onMounted(() => {
  if (!CSS.supports("view-timeline", "--test")) {
    console.log("Falling back to IntersectionObserver...");
    observer.value = useStickyIntersectionObserver(76);
  }
});

const [focused, tabKeydown] = useRovingTabIndex(9, 3);
const keydown = (e: KeyboardEvent) => {
  let sibling;

  const row = getStopParent(document.activeElement);
  if (e.key === "ArrowUp") {
    sibling = row?.previousElementSibling;
    while (sibling && !sibling.classList.contains("stop")) {
      sibling = sibling?.previousElementSibling;
    }
  } else if (e.key === "ArrowDown") {
    sibling = row?.nextElementSibling;
    while (sibling && !sibling.classList.contains("stop")) {
      sibling = sibling?.nextElementSibling;
    }
  }

  if (sibling) {
    // @ts-ignore
    sibling.querySelector("[tabindex='0']")?.focus();
    e.preventDefault();
  } else {
    tabKeydown(e);
  }
};

const toast = useToast();
const copy = async (id: number): Promise<void> => {
  const puzzles = store.puzzlesByRound.get(id);
  if (!puzzles) return;
  const data = puzzles.map((p) =>
    [p.name, p.answer || `${StatusEmoji(p.status)} ${StatusLabel(p.status)}`]);
  const text = data.map(([a, b]) => `${a}, ${b}`).join("\n");
  const html = "<table>\n" + data.map(([a, b]) =>
    `<tr><td>${a}</td><td>${b}</td></tr>`
  ).join("\n") + "</table>";
  if (navigator.clipboard.write) {
    await navigator.clipboard.write([
      new ClipboardItem({
        "text/plain": new Blob([text], { type: "text/plain" }),
        "text/html": new Blob([html], { type: "text/html" }),
      }),
    ]);
  } else {
    await navigator.clipboard.writeText(text);
  }
  toast.add({
    title: "Copied",
    color: "green",
    icon: "i-heroicons-clipboard-document-check",
  });
};

const welcome = ref();
const editing = ref<
  { kind: "round" | "puzzle" | "admin"; id?: void; } |
  { kind: "round" | "puzzle", id: number; }
>();
const discord = ref(false);
onMounted(() => discord.value = !!localStorage.getItem("discord"));
const click = (kind: "round" | "puzzle" | "admin") => {
  if (editing.value?.kind === kind && !editing.value.id) {
    editing.value = undefined;
  } else {
    editing.value = { kind };
  }
};
const toggle = () => {
  discord.value = !discord.value;
  if (discord.value) localStorage.setItem("discord", "*");
  else localStorage.removeItem("discord");
};
const close = () => {
  if (editing.value && !editing.value.id) {
    welcome.value?.focus(editing.value.kind);
  }
  editing.value = undefined;
};

// If an input element is focused, keydown events bubble from the input.
// Otherwise, they bubble from the body.
onMounted(() => window.addEventListener("keydown",
  (e) => (e.key === "Escape") && close()));
</script>

<template>
  <MainHeader :rounds="store.rounds" :observer="observer" :connected="connected" />
  <main @keydown="keydown">
    <div class="rule first"></div>
    <div class="rule"></div>
    <div class="rule"></div>
    <template v-for="[i, round] of store.rounds.entries()">
      <RoundHeader :round="round" :timeline="timelineFromID(i)"
        :next-timeline="nextTimelineFromID(i)" :observer="observer"
        @copy="() => copy(round.id)"
        @edit="() => (editing = { kind: 'round', id: round.id })" />
      <Puzzle v-for="puzzle in store.puzzlesByRound.get(round.id)" :puzzle="puzzle"
        :round="round" :discord="discord" :focused="focused"
        @edit="() => (editing = { kind: 'puzzle', id: puzzle.id })" />
      <div class="empty" v-if="!round.total">
        🫙&hairsp; No Puzzles
      </div>
      <hr>
    </template>
    <WelcomeAndAdminBar ref="welcome" :discord="discord" @click="click"
      @toggle="toggle" />
    <Modal v-if="!!editing" @close="close">
      <AdminForm v-if="editing.kind === 'admin'" @close=close />
      <AddRoundPuzzleForm v-else-if="!editing.id" :kind="editing.kind" @close="close" />
      <EditRoundForm v-else-if="editing?.kind === 'round'" :id="editing.id"
        @close="close" />
      <EditPuzzleForm v-else-if="editing?.kind === 'puzzle'" :id="editing.id"
        @close="close" />
    </Modal>
  </main>
</template>

<style scoped>
/* Layout */
main {
  padding: calc(6rem - 1.4rem) 0.5vw 0.5rem 2vw;
  min-width: 1024px;
  display: grid;
  grid-template-columns: 8rem 6fr 6fr 4fr 8fr;
  column-gap: 0.66rem;
}

.rule {
  width: 0;
  height: calc(100vh - 6rem - 1px);
  position: sticky;
  top: calc(6rem + 1px);
  margin-bottom: -100vh;

  margin-left: -0.33rem;
  border-left: 1px solid oklch(95% 0.03 275deg);

  z-index: 12;
}

.rule.first {
  grid-column: 3;
}

hr {
  grid-column: 1 / 6;

  margin: 0.5rem -0.5vw 0.5rem -2vw;
  border-bottom: 1px solid oklch(90% 0.03 275deg);
}

.empty {
  grid-column: 1 / 3;
  margin: 0 1.5rem;
}

/* Theming */
.empty {
  font-size: 0.9rem;
  opacity: 50%;
  user-select: none;
}

/* Animation */
main {
  timeline-scope: v-bind(timelines);
}
</style>
