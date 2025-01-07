<script setup lang="ts">
const props = defineProps<{
  puzzle: Puzzle; round: AnnotatedRound; focused: FocusInfo;
}>();
const emit = defineEmits<{ (e: "edit"): void; }>();

const row = useTemplateRef("row");
defineExpose({
  id: props.puzzle.id,
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
      <PuzzleName :puzzle="puzzle" :round="round"
        @focusin="() => (focused.index !== 4) && (focused.index = 3)"
        @edit="() => emit('edit')" />
      <PuzzleStatus :puzzle="puzzle" :round="round"
        @focusin="() => (focused.index !== 6) && (focused.index = 5)" />
      <PuzzleNoteLocation :puzzle="puzzle" field="location" :tabsequence="7"
        @focusin="() => (focused.index = 7)" />
      <PuzzleNoteLocation :puzzle="puzzle" field="note" :tabsequence="8"
        @focusin="() => (focused.index = 8)" />
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
