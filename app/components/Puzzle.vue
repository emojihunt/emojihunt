<script setup lang="ts">
const props = defineProps<{
  puzzle: Puzzle; round: AnnotatedRound; focused: FocusInfo;
}>();
const emit = defineEmits<{ (e: "edit"): void; }>();

const focus = () => nextTick(() =>
  row.value?.querySelector<HTMLElement>("[tabindex='0']")?.focus());
defineExpose({
  id: props.puzzle.id, focus
});

const row = useTemplateRef("row");
const keydown = (e: KeyboardEvent) => {
  if (e.key === "ArrowRight") {
    if (props.focused.index < 8) props.focused.index += 1;
  } else if (e.key === "ArrowLeft") {
    if (props.focused.index > 0) props.focused.index -= 1;
  } else {
    return;
  }
  focus();
  e.preventDefault();
  e.stopPropagation();
};
</script>

<template>
  <span ref="row" class="puzzle" :data-puzzle="puzzle.id" @keydown="keydown">
    <PuzzleButtons :puzzle="puzzle" :focused="focused" />
    <span class="data">
      <PuzzleName :puzzle="puzzle" :round="round" :focused="focused"
        @focusin="() => (focused.index !== 4) && (focused.index = 3)"
        @edit="() => emit('edit')" />
      <PuzzleStatus :puzzle="puzzle" :round="round" :focused="focused"
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
