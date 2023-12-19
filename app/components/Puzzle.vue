<script setup lang="ts">
const props = defineProps<{
  puzzle: Puzzle; round: RoundStats; focused: FocusInfo;
}>();
</script>

<template>
  <span class="row stop">
    <PuzzleButtons :puzzle="puzzle" :focused="focused" />
    <span class="data">
      <PuzzleCell :puzzle="puzzle" field="name" readonly :tabindex="tabIndex(focused, 3)" />
      <PuzzleStatus :puzzle="puzzle" :tabindex="tabIndex(focused, 4)" />
      <PuzzleCell :puzzle="puzzle" field="location" :tabindex="tabIndex(focused, 5)" />
      <PuzzleCell :puzzle="puzzle" field="note" :tabindex="tabIndex(focused, 6)" />
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
}

/* Theming */
.data {
  border-top: 1px solid transparent;
  border-bottom: 1px solid transparent;
}

.row:hover .data {
  border-color: oklch(86% 0 0deg);
}
</style>
