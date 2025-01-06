<script setup lang="ts">
useHead({
  htmlAttrs: { lang: "en" },
  title: "Puzzle Tracker",
});
const store = usePuzzles();
await store.refresh();
const connected = useAbly();

// It doesn't look great when the round headers stack up on top of one another.
// We want each round header to disappear when it's covered by the next. Use CSS
// scroll-linked animations if supported and fall back to IntersectionObserver
// if not.
const timelines = computed(() => store.rounds.map((_, i) => timelineFromSequence(i)));
const observer = ref<IntersectionObserver>();
onMounted(() => {
  if (!CSS.supports("view-timeline", "--test")) {
    console.log("Falling back to IntersectionObserver...");
    observer.value = useStickyIntersectionObserver(76);
  }
});

// We don't update the hash, so strip it from the URL on page load.
onMounted(() => document.location.hash && history.pushState(
  "", document.title, window.location.pathname + window.location.search,
));

const welcome = useTemplateRef("welcome");
const editing = ref<
  { kind: "round" | "puzzle" | "admin"; id?: void; } |
  { kind: "round" | "puzzle", id: number; }
>();
const click = (kind: "round" | "puzzle" | "admin") => {
  if (editing.value?.kind === kind && !editing.value.id) {
    editing.value = undefined;
  } else {
    editing.value = { kind };
  }
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

const showNav = computed(() => store.puzzleCount >= 42);
const navMargin = computed(() => store.puzzleCount >= 42 ? "3.5rem" : "2vw");
</script>

<template>
  <MainHeader :connected="!!connected" />
  <main>
    <EmojiNav v-if="showNav" :observer="observer" />
    <div class="rule first"></div>
    <div class="rule"></div>
    <div class="rule"></div>
    <RoundsAndPuzzles :observer="observer"
      @edit="(kind, id) => { editing = { kind, id }; }" />
    <WelcomeAndAdminBar ref="welcome" @click="click" />
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
  --nav-margin: v-bind(navMargin);

  padding: var(--header-stop) 0.5vw 0.5rem var(--nav-margin);
  min-width: 1024px;
  display: grid;
  grid-template-columns: 8rem 6fr 5fr 4fr 8fr;
  column-gap: 0.66rem;
}

.rule {
  width: 0;
  height: calc(100vh - var(--header-height-outer));
  position: sticky;
  top: var(--header-height-outer);
  margin: 0 0 -100vh -0.33rem;

  z-index: 12;
}

.rule.first {
  grid-column: 3;
}

/* Theming */
.rule {
  border-right: 1px solid oklch(95% 0.03 275deg);
}

/* Animation */
main {
  timeline-scope: v-bind(timelines);
}
</style>
