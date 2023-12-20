<script setup lang="ts">
useHead({ title: "Puzzle Tracker" });
const store = usePuzzles();
await store.refresh();

// Puzzle & Round Helpers
const timelineFromID = (id: number) => `--round-${id}`;
const nextTimelineFromID = (id: number): string | undefined =>
  store.rounds[id + 1] ? timelineFromID(id + 1) : undefined;

// It doesn't look great when the round headers stack up on top of one another.
// We want each round header to disappear when it's covered by the next. Use CSS
// scroll-linked animations if supported and fall back to IntersectionObserver
// if not.
const timelines = store.rounds.map((_, i) => timelineFromID(i));
let observer: IntersectionObserver | undefined;
if (import.meta.client && !CSS.supports("view-timeline", "--test")) {
  console.log("Falling back to IntersectionObserver...");
  observer = useStickyIntersectionObserver(76);
}

const [focused, tabKeydown] = useRovingTabIndex(7, 3);
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
</script>

<template>
  <MainHeader :rounds="store.rounds" :observer="observer" />
  <main @keydown="keydown">
    <div class="rule first"></div>
    <div class="rule"></div>
    <div class="rule"></div>
    <template v-for="[i, round] of store.rounds.entries()">
      <RoundHeader :round="round" :timeline="timelineFromID(i)"
        :next-timeline="nextTimelineFromID(i)" :observer="observer" />
      <Puzzle v-for="puzzle in store.puzzles.get(round.id)" :puzzle="puzzle" :round="round"
        :focused="focused" />
      <div class="empty" v-if="!round.total">
        🫙&hairsp; No Puzzles
      </div>
      <hr>
    </template>
    <WelcomeAndAdminBar />
  </main>
</template>

<style scoped>
/* Layout */
main {
  padding: calc(6rem - 1.4rem) 0.5vw 18vh 2vw;
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
