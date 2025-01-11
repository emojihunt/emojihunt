<script setup lang="ts">
const { puzzle } = defineProps<{ puzzle: Puzzle; }>();
const emit = defineEmits<{ (e: "edit"): void; }>();

const row = useTemplateRef("row");
defineExpose({
  id: puzzle.id,
  focus() {
    nextTick(() =>
      row.value?.querySelector<HTMLElement>("[tabindex='0']")?.focus());
  }
});
</script>

<template>
  <span ref="row" class="puzzle" :data-puzzle="puzzle.id">
    <PuzzleButtons :puzzle="puzzle" />
    <span class="data">
      <PuzzleName :puzzle="puzzle" @edit="() => emit('edit')" />
      <PuzzleStatus :puzzle="puzzle" />
      <PuzzleNoteLocation :puzzle="puzzle" field="location" :tabsequence="7" />
      <PuzzleNoteLocation :puzzle="puzzle" field="note" :tabsequence="8" />
    </span>
  </span>
</template>

<style scoped>
/* Layout */
.puzzle {
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

.puzzle:hover .data {
  border-color: oklch(86% 0 0deg);
}
</style>
