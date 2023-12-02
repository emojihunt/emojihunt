<script setup lang="ts">
useHead({ title: "Puzzle Tracker" });
const data = await useAPI("/puzzles");

// HACK: apply hard-coded colors to rounds for testing
const hues: { [round: string]: number; } = {
  "1": 241, "2": 178, "3": 80, "4": 45,
  "5": 255, "6": 19, "7": 69, "8": 205,
  "9": 28, "10": 24, "11": 141,
};

// Group puzzles by round
const puzzles: { [round: string]: Puzzle[]; } = {};
for (const puzzle of data.value) {
  const id = puzzle.round.id;
  puzzles[id] ||= [];
  puzzles[id].push(puzzle);
}

// Compute round stats
const rounds: { [round: string]: RoundStats; } = {};
for (const id of Object.keys(puzzles)) {
  const example = puzzles[id][0].round;
  rounds[id] = {
    anchor: example.name.trim().toLowerCase().replaceAll(" ", "-"),
    complete: puzzles[id].filter((p => !p.answer)).length == 0,
    hue: hues[id],
    solved: puzzles[id].filter((p) => !!p.answer).length,
    total: puzzles[id].length,

    id: example.id,
    name: example.name.trim(),
    emoji: example.emoji,
  };
}

// Puzzle & Round Helpers
const timelineFromID = (id: string) => `--round-${id}`;
const nextTimelineFromID = (id: string): string | undefined => {
  const i = (parseInt(id) + 1).toString();
  return puzzles[i] ? timelineFromID(i) : undefined;
};

// It doesn't look great when the round headers stack up on top of one another.
// We want each round header to disappear when it's covered by the next. Use CSS
// scroll-linked animations if supported and fall back to IntersectionObserver
// if not.
const timelines = Object.keys(puzzles).map(timelineFromID);
let observer: IntersectionObserver | undefined;
if (import.meta.client && !CSS.supports("view-timeline", "--test")) {
  console.log("Falling back to IntersectionObserver...");
  observer = useStickyIntersectionObserver(74);
}
</script>

<template>
  <Navbar :rounds="Object.values(rounds)" :observer="observer" />
  <main>
    <div class="rule first"></div>
    <div class="rule"></div>
    <div class="rule"></div>
    <template v-for="id of Object.keys(puzzles)">
      <PuzzleListHeader :round="rounds[id]" :timeline="timelineFromID(id)"
        :next-timeline="nextTimelineFromID(id)" :observer="observer" />
      <PuzzleListRow v-for="puzzle in puzzles[id]" :puzzle="puzzle" :round="rounds[id]" />
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
  border-left: 1px solid oklch(96% 0.01 286deg);

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

  border-bottom: 1px solid oklch(96% 0.01 286deg);
  z-index: -10;
}

/* Animation */
main {
  timeline-scope: v-bind(timelines);
}
</style>
