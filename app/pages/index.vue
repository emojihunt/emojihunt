<script setup lang="ts">
useHead({
  htmlAttrs: { lang: "en" },
  title: "Puzzle Tracker",
  meta: [
    { name: "viewport", content: "width=device-width, initial-scale=1, viewport-fit=cover" },
    { name: "theme-color", content: "oklch(30% 0 0deg)" },
  ]
});
const store = usePuzzles();
await store.refresh();
const connected = useAbly();

// It doesn't look great when the round headers stack up on top of one another.
// We want each round header to disappear when it's covered by the next. Use CSS
// scroll-linked animations if supported and fall back to IntersectionObserver
// if not.
const timelines = [...Array(50).keys()].map((i) => timelineFromSequence(i));
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

const header = useTemplateRef<any>("header");
const filter = computed(() => header.value?.filter);

const table = useTemplateRef("table");
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
  if (editing.value === undefined) {
    return;
  } else if (!editing.value.id) {
    welcome.value?.focus(editing.value.kind);
  } else if (editing.value.kind === "puzzle") {
    table.value?.focus(editing.value.id);
  }
  editing.value = undefined;
};

// If an input element is focused, keydown events bubble from the input.
// Otherwise, they bubble from the body.
onMounted(() => window.addEventListener("keydown",
  (e) => (e.key === "Escape") && close()));

const showNav = computed(() => store.puzzleCount >= 42);
const navMargin = computed(() => store.puzzleCount >= 42 ? "4.5rem" : "2vw");
</script>

<template>
  <MainHeader ref="header" :connected="!!connected" />
  <div :class="['content', filter && 'filter']">
    <EmojiNav v-if="showNav" :filter="!!filter" :observer="observer"
      @navigate="() => table?.navigate()" />
    <div class="rule first"></div>
    <div class="rule"></div>
    <div class="rule"></div>
    <RoundsAndPuzzles ref="table" :filter="!!filter" :observer="observer"
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
  </div>
</template>

<style scoped>
/* Layout */
.content {
  --nav-margin: v-bind(navMargin);

  padding: var(--header-stop) 0.5vw 0.5rem calc(env(safe-area-inset-left) + var(--nav-margin));
  min-width: 1024px;
  display: grid;
  grid-template-columns: 8rem 6fr 5fr 4fr 8fr;
  column-gap: 0.66rem;
}

.rule {
  width: 0;
  height: calc(100dvh - var(--header-height-outer));
  position: sticky;
  top: var(--header-height-outer);
  margin: 0 0 -100dvh -0.33rem;

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
.content {
  timeline-scope: v-bind(timelines);
}

/* Media Queries */
@media (max-width: 768px) {
  .content {
    padding-left: 2vw;
  }

  nav {
    display: none;
  }
}
</style>
