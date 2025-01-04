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
const timelineFromID = (id: number) => `--round-${id}`;
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

const welcome = ref();
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
</script>

<template>
  <MainHeader :rounds="store.rounds" :observer="observer" :connected="connected" />
  <main @keydown="keydown">
    <div class="rule first"></div>
    <div class="rule"></div>
    <div class="rule"></div>
    <RoundAndPuzzles v-for="[i, round] of store.rounds.entries()" :round="round" :i="i"
      :focused="focused" :observer="observer"
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
  padding: calc(6rem - 1.4rem) 0.5vw 0.5rem 2vw;
  min-width: 1024px;
  display: grid;
  grid-template-columns: 8rem 6fr 5fr 4fr 8fr;
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

/* Animation */
main {
  timeline-scope: v-bind(timelines);
}
</style>
