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
  <main>
    <template v-for="id in Object.keys(puzzles)">
      <PuzzleListHeader :puzzles="puzzles[id as any]" />
      <PuzzleListRow v-for="puzzle in puzzles[id as any]" :puzzle="puzzle" />
    </template>
  </main>
</template>

<style scoped>
main {
  padding: 1vh 4vw 5vh;
  min-width: 75rem;
  display: grid;
  grid-template-columns: 4rem 8fr 8rem 8fr 12fr;
}
</style>
