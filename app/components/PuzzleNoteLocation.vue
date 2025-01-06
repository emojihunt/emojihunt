<script setup lang="ts">
const props = defineProps<{
  puzzle: Puzzle;
  field: "location" | "note";
  tabindex: number;
}>();
const store = usePuzzles();
const saving = ref(false);

const tooltip = computed(() => {
  if (props.field === "location") {
    const id = props.puzzle.voice_room;
    if (!id) return;
    const channel = store.voiceRooms[id];
    if (!channel) return;
    // We expect the channel's emoji to go at the end
    const p = channel.split(" ");
    if ([...p[p.length - 1]].length === 1) {
      const text = p.slice(0, p.length - 1).join(" ");
      return {
        emoji: p[p.length - 1],
        placeholder: text,
        text: `in ${text}`,
      };
    } else {
      return { emoji: "📻", placeholder: channel, text: `in ${channel}` };
    }
  } else {
    const reminder = parseReminder(props.puzzle);
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
  store.updatePuzzleOptimistic(props.puzzle.id, { [props.field]: updated })
    .finally(() => (saving.value = false));
};
</script>

<template>
  <div class="cell" :class="field">
    <ETooltip v-if="tooltip" :text="tooltip.text">
      <span class="emoji">{{ tooltip.emoji }}</span>
    </ETooltip>
    <EditableSpan :value="puzzle[field]" :tabindex="tabindex" @save="save"
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
  top: calc((1.8rem - 1em) / 2);
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
