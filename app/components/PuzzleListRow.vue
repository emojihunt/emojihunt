<script setup lang="ts">
const props = defineProps<{
  puzzle: Puzzle; round: RoundStats; focused: FocusInfo;
}>();
</script>

<template>
  <span class="row stop">
    <PuzzleListButtons :puzzle="puzzle" :focused="focused" />
    <span class="data">
      <PuzzleListCell :puzzle="puzzle" field="name" :hue="round.hue" :style="'thick'"
        readonly :tabindex="tabIndex(focused, 3)" />
      <PuzzleListStatus :puzzle="puzzle" :tabindex="tabIndex(focused, 4)" />
      <PuzzleListCell :puzzle="puzzle" field="location" :style="'thin'"
        :tabindex="tabIndex(focused, 5)" />
      <PuzzleListCell :puzzle="puzzle" field="description" :style="'thin'"
        :tabindex="tabIndex(focused, 6)" />
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
