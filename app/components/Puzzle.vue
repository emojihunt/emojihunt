<script setup lang="ts">
const { id } = defineProps<{ id: number; }>();
const emit = defineEmits<{ (e: "edit"): void; }>();

const { puzzles, rounds } = usePuzzles();
const puzzle = puzzles.get(id)!;
const hue = computed(() => rounds.get(puzzle.round)?.hue || 0);

// Check if this round has only metas (so we don't filter them out)
const roundHasOnlyMetas = computed(() => {
  const roundPuzzles = [...puzzles.values()].filter(p => p.round === puzzle.round);
  return roundPuzzles.length > 0 && roundPuzzles.every(p => p.meta);
});

// Puzzle is filterable (hidden when priority mode is on) if:
// - It's a task (name starts with '[Task] '), OR
// - It has an answer (is solved), unless it's a meta in a round with only metas
const isFilterable = computed(() => {
  if (puzzle.name.startsWith('[Task] ')) return true;
  if (puzzle.answer && puzzle.meta && roundHasOnlyMetas.value) return false;
  return !!puzzle.answer;
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
