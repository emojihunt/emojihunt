<script setup lang="ts">
const props = defineProps<{
  puzzle: Puzzle; round: AnnotatedRound; focused: FocusInfo;
}>();
const emit = defineEmits<{ (e: "edit"): void; }>();
</script>

<template>
  <span class="row stop">
    <PuzzleButtons :puzzle="puzzle" :focused="focused" />
    <span class="data">
      <PuzzleName :puzzle="puzzle" :round="round" :focused="focused"
        @focusin="() => (focused.index !== 4) && (focused.index = 3)"
        @edit="() => emit('edit')" />
      <PuzzleStatus :puzzle="puzzle" :focused="focused"
        @focusin="() => (focused.index !== 6) && (focused.index = 5)" />
      <PuzzleNoteLocation :puzzle="puzzle" field="location"
        :tabindex="tabIndex(focused, 7)" @focusin="() => (focused.index = 7)" />
      <PuzzleNoteLocation :puzzle="puzzle" field="note" :tabindex="tabIndex(focused, 8)"
        @focusin="() => (focused.index = 8)" />
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
