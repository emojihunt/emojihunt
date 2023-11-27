<script setup lang="ts">
const data = await useAPI("/puzzles");

// HACK: apply hard-coded colors to rounds for testing
const colors = {
  1: 241, 2: 178, 3: 80, 4: 45,
  5: 255, 6: 19, 7: 69, 8: 205,
  9: 28, 10: 24, 11: 141,
};
for (const puzzle of data.value) {
  const id = puzzle.round.id;
  // @ts-ignore
  puzzle.round.color = colors[id];
}

// Group puzzles by round
const puzzles: { [round: number]: any; } = {};
for (const puzzle of data.value) {
  const id: number = puzzle.round.id;
  puzzles[id] ||= [];
  puzzles[id].push(puzzle);
}
</script>

<template>
  <header></header>
  <main>
    <template v-for="id in Object.keys(puzzles)">
      <PuzzleListHeader :puzzles="puzzles[id as any]" />
      <PuzzleListRow v-for="puzzle in puzzles[id as any]" :puzzle="puzzle" />
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
</style>
