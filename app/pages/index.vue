<script setup lang="ts">
const data = await useAPI("/puzzles");

// HACK: apply hard-coded colors to rounds for testing
const hues: { [round: string]: number; } = {
  "1": 241, "2": 178, "3": 80, "4": 45,
  "5": 255, "6": 19, "7": 69, "8": 205,
  "9": 28, "10": 24, "11": 141,
};

// Group puzzles by round
const puzzles: { [round: string]: any; } = {};
for (const puzzle of data.value) {
  const id = puzzle.round.id;
  puzzles[id] ||= [];
  puzzles[id].push(puzzle);
}

// It doesn't look great when the round headers stack up on top of one another.
// We want each round header to disappear when it's covered by the next. Use CSS
// scroll-linked animations if supported and fall back to IntersectionObserver
// if not.
const timelines = Object.keys(puzzles).map((id) => `--round-${id}`);
const observer = import.meta.client && CSS.supports("view-timeline", "--test") ?
  undefined : useStickyIntersectionObserver(74);
</script>

<template>
  <header></header>
  <main>
    <template v-for="id in Object.keys(puzzles)">
      <PuzzleListHeader :puzzles="puzzles[id]" :hue="hues[id]" :timeline="`--round-${id}`"
        :next-timeline="!!puzzles[(parseInt(id) + 1)] ? `--round-${parseInt(id) + 1}` : undefined" :observer="observer" />
      <PuzzleListRow v-for="puzzle in puzzles[id]" :puzzle="puzzle" />
    </template>
  </main>
</template>

<style scoped>
/* Layout */
header {
  width: 100%;
  height: 6rem;
  position: fixed;
}

main {
  padding: 2rem 1rem 16vh 2rem;
  min-width: 75rem;
  display: grid;
  grid-template-columns: 8rem 6fr 6fr 4fr 8fr;
}

/* Themeing */
header {
  background-color: oklch(98% 0.01 286deg);
  border-bottom: 1px solid oklch(80% 0.01 286deg);
  filter: drop-shadow(0 1.5rem 1rem oklch(100% 0 0deg));
}

/* Animation */
main {
  timeline-scope: v-bind(timelines);
}
</style>
