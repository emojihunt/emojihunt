<script setup lang="ts">
useHead({ title: "Puzzle Tracker" });
const store = usePuzzles();
await store.refresh();

// Puzzle & Round Helpers
const timelineFromID = (id: string) => `--round-${id}`;
const nextTimelineFromID = (id: string): string | undefined => {
  const i = (parseInt(id) + 1).toString();
  return store.puzzlesByRound[i] ? timelineFromID(i) : undefined;
};

// It doesn't look great when the round headers stack up on top of one another.
// We want each round header to disappear when it's covered by the next. Use CSS
// scroll-linked animations if supported and fall back to IntersectionObserver
// if not.
const timelines = Object.keys(store.puzzlesByRound).map(timelineFromID);
let observer: IntersectionObserver | undefined;
if (import.meta.client && !CSS.supports("view-timeline", "--test")) {
  console.log("Falling back to IntersectionObserver...");
  observer = useStickyIntersectionObserver(76);
}

const [focused, tabKeydown] = useRovingTabIndex(7, 3);
const keydown = (e: KeyboardEvent) => {
  let sibling;

  const row = getStopParent(document.activeElement);
  if (e.key == "ArrowUp") {
    sibling = row?.previousElementSibling;
    while (sibling && !sibling.classList.contains("stop")) {
      sibling = sibling?.previousElementSibling;
    }
  } else if (e.key == "ArrowDown") {
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
</script>

<template>
  <MainHeader :rounds="Object.values(store.roundStats)" :observer="observer" />
  <main @keydown="keydown">
    <div class="rule first"></div>
    <div class="rule"></div>
    <div class="rule"></div>
    <template v-for="id of Object.keys(store.puzzlesByRound)">
      <RoundHeader :round="store.roundStats[id]" :timeline="timelineFromID(id)"
        :next-timeline="nextTimelineFromID(id)" :observer="observer" />
      <Puzzle v-for="puzzle in store.puzzlesByRound[id]" :puzzle="puzzle"
        :round="store.roundStats[id]" :focused="focused" />
      <hr>
    </template>
  </main>
</template>

<style scoped>
/* Layout */
main {
  padding: calc(6rem - 1.4rem - 2.8rem) 0.5vw 18vh 2vw;
  min-width: 75rem;
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
  width: 100vw;
  height: 0;

  position: relative;
  left: -2rem;
  top: 0.5rem;

  border-bottom: 1px solid oklch(90% 0.03 275deg);
  z-index: -10;
}

/* Animation */
main {
  timeline-scope: v-bind(timelines);
}
</style>
