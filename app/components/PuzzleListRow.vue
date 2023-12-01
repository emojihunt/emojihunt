<script setup lang="ts">
const props = defineProps<{ puzzle: Puzzle; round: RoundStats; }>();
</script>

<template>
  <span class="row">
    <PuzzleButtons :puzzle="puzzle" />
    <span class="data">
      <PuzzleListCell :content="puzzle.name" :hue="round.hue" :style="'thick'" />
      <PuzzleListStatus :puzzle="puzzle" />
      <PuzzleListCell :content="puzzle.location" :style="'thin'" />
      <PuzzleListCell :content="puzzle.description" :style="'thin'" />
    </span>
  </span>
</template>

<style scoped>
/* Layout */
.row {
  grid-column: 1 / 6;
  display: grid;
  grid-template-columns: subgrid;
}

.data {
  grid-column: 2 / 6;
  display: grid;
  grid-template-columns: subgrid;

  padding-top: 2px;
}

/* Themeing */
.data {
  border-top: 1px solid transparent;
  border-bottom: 1px solid transparent;
}

.row:hover .data {
  border-color: oklch(86% 0 0deg);
}
</style>
