<script setup lang="ts">
const { id } = defineProps<{ id: number; }>();
const emit = defineEmits<{ (e: "edit"): void; }>();

const { puzzles, rounds } = usePuzzles();
const puzzle = puzzles.get(id)!;
const hue = computed(() => rounds.get(puzzle.round)?.hue || 0);

// Puzzle is filterable (hidden when priority mode is on) if:
// - It's a task (name starts with '[Task] '), OR
// - It's solved (has an answer)
const isFilterable = computed(() => {
  return puzzle.name.startsWith('[Task] ') || !!puzzle.answer;
});

const row = useTemplateRef("row");
defineExpose({
  id,
  focus() {
    nextTick(() =>
      row.value?.querySelector<HTMLElement>("[tabindex='0']")?.focus());
  },
  isVisible(): boolean {
    return !!row.value?.checkVisibility();
  },
});
</script>

<template>
  <span ref="row" class="puzzle" :data-puzzle="id" :class="isFilterable && 'filterable'">
    <PuzzleButtons :id />
    <span class="data">
      <PuzzleName :id @edit="() => emit('edit')" />
      <PuzzleStatus :id />
      <PuzzleLocation :id field="location" />
      <PuzzleNote :id field="note" />
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
.puzzle {
  --round-hue: v-bind(hue);
}

.data {
  border-top: 1px solid transparent;
  border-bottom: 1px solid transparent;
}

.puzzle:hover .data {
  border-color: oklch(86% 0 0deg);
}

:global(.filter .filterable) {
  display: none;
}
</style>
