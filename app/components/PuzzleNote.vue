<script setup lang="ts">
const { id } = defineProps<{ id: number; }>();

const { puzzles, voiceRooms, updatePuzzleOptimistic } = usePuzzles();
const puzzle = puzzles.get(id)!;
const saving = ref(false);

const tooltip = computed(() => {
  const reminder = parseTimestamp(puzzle.reminder);
  if (!reminder) return;
  const formatted = reminder.toLocaleString("en-US", {
    weekday: "long",
    hour: "numeric",
    minute: "2-digit",
    timeZone: "America/New_York"
  });
  return { emoji: "⏰", text: `${formatted} Boston Time` };
});

const save = (updated: string) => {
  saving.value = true;
  updatePuzzleOptimistic(id, { note: updated })
    .finally(() => (saving.value = false));
};
</script>

<template>
  <div class="cell">
    <ETooltip v-if="tooltip" :text="tooltip.text">
      <span class="emoji">{{ tooltip.emoji }}</span>
    </ETooltip>
    <EditableSpan :value="puzzle.note" :tabsequence="8" @save="save" />
    <Spinner v-if="saving" class="spinner" />
  </div>
</template>

<style scoped>
/* Layout */
.emoji {
  width: 1.5rem;
  text-align: center;
}

.spinner {
  right: 0.33rem;
  top: 6px;
}

/* Theming */
.cell {
  font-weight: 400;
  font-size: 0.8125rem;
}

.emoji {
  line-height: 1.75rem;
  filter: opacity(90%);

  cursor: default;
  user-select: none;
}

.emoji:hover {
  transform: scale(110%);
  filter: drop-shadow(0 1px 1px oklch(85% 0 0deg));
}
</style>
