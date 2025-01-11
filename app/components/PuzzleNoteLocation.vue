<script setup lang="ts">
const { puzzle, field, tabsequence } = defineProps<{
  puzzle: Puzzle;
  field: "location" | "note";
  tabsequence: number;
}>();
const store = usePuzzles();
const saving = ref(false);

const tooltip = computed(() => {
  if (field === "location") {
    const id = puzzle.voice_room;
    if (!id) return;
    const channel = store.voiceRooms.get(id);
    if (!channel) return;
    return { emoji: channel.emoji, placeholder: channel.name, text: `in ${channel.name}` };
  } else {
    const reminder = parseReminder(puzzle);
    if (!reminder) return;
    const formatted = reminder.toLocaleString("en-US", {
      weekday: "long",
      hour: "numeric",
      minute: "2-digit",
      timeZone: "America/New_York"
    });
    return { emoji: "⏰", text: `${formatted} Boston Time` };
  }
});

const save = (updated: string) => {
  saving.value = true;
  store.updatePuzzleOptimistic(puzzle.id, { [field]: updated })
    .finally(() => (saving.value = false));
};
</script>

<template>
  <div class="cell" :class="field">
    <ETooltip v-if="tooltip" :text="tooltip.text">
      <span class="emoji">{{ tooltip.emoji }}</span>
    </ETooltip>
    <EditableSpan :value="puzzle[field]" :tabsequence="tabsequence" @save="save"
      :placeholder="tooltip?.placeholder" />
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
