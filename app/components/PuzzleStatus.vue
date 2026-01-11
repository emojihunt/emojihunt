<script setup lang="ts">
const { id } = defineProps<{ id: number; }>();
const toast = useToast();

const { puzzles, updatePuzzleOptimistic } = usePuzzles();
const puzzle = puzzles.get(id)!;

const input = useTemplateRef("input");
const button = useTemplateRef("button");

const saving = ref(false);
const answering = ref<Status | null>(null);
const expanded = inject(ExpandedKey);

const save = (answer: string) => {
  if (!answer) return;  // cannot be blank
  answer = answer.toUpperCase();
  if (answering.value) {
    // Answer with state change to "Solved", etc.
    saving.value = true;
    updatePuzzleOptimistic(id, { answer, status: answering.value, voice_room: "" })
      .catch(() => toast.add({
        title: "Error", color: "error", description: "Failed to save puzzle",
        icon: "i-heroicons-exclamation-triangle",
      }))
      .finally(() => (saving.value = false));
    answering.value = null;
  } else {
    // Regular answer fixup
    saving.value = true;
    updatePuzzleOptimistic(id, { answer })
      .catch(() => toast.add({
        title: "Error", color: "error", description: "Failed to save puzzle",
        icon: "i-heroicons-exclamation-triangle",
      }))
      .finally(() => (saving.value = false));
  }
};
const cancel = () => answering.value && (answering.value = null, expanded && (expanded.value = 0));

const items = Object.values(Status).map(
  (s) => ({ id: s, emoji: StatusEmoji(s), name: StatusLabel(s), right: StatusNeedsAnswer(s) })
);
const select = (status: Status) => {
  if (expanded) expanded.value = 0;
  answering.value = null;
  if (!StatusNeedsAnswer(status)) {
    saving.value = true;
    updatePuzzleOptimistic(id, { status, answer: "" })
      .catch(() => toast.add({
        title: "Error", color: "error", description: "Failed to save puzzle",
        icon: "i-heroicons-exclamation-triangle",
      }))
      .finally(() => (saving.value = false));
    nextTick(() => button.value?.focus());
  } else if (puzzle.answer) {
    saving.value = true;
    updatePuzzleOptimistic(id, { status })
      .catch(() => toast.add({
        title: "Error", color: "error", description: "Failed to save puzzle",
        icon: "i-heroicons-exclamation-triangle",
      }))
      .finally(() => (saving.value = false));
  } else {
    answering.value = status;
    nextTick(() => input.value?.focus());
  }
};
</script>

<template>
  <div class="cell">
    <div class="row">
      <EditableSpan v-if="puzzle.answer || answering" ref="input" class="answer"
        :value="puzzle.answer" :tabsequence="5" :sticky="!!answering" @save="save"
        @cancel="cancel" :readonly="saving" />
      <button v-else ref="button" class="status" :data-tabsequence="56"
        @click="() => expanded = (expanded === id ? 0 : id)">
        <span class="highlight" :class="puzzle.meta && 'meta'">
          {{ StatusEmoji(puzzle.status) }} {{ StatusLabel(puzzle.status) }}
        </span>
      </button>

      <Spinner v-if="saving" />

      <ETooltip v-if="puzzle.answer || answering" :text="answering || puzzle.status"
        side="left" class="right">
        <button :data-tabsequence="6" class="solve"
          @click="() => answering ? (answering = null, expanded = id) : expanded = (expanded === id ? 0 : id)">
          {{ StatusEmoji(answering || puzzle.status) }}
        </button>
      </ETooltip>
      <ETooltip v-else-if="false" text="Last edit: ..." side="left" class="right">
        <div class="sheet-active"></div>
        <div class="sheet-recent">30</div>
      </ETooltip>
    </div>
    <div v-if="answering" class="hint">🎉 Press Enter to record answer</div>
    <OptionPane v-if="expanded === id" :items="items" double @select="select" />
  </div>
</template>

<style scoped>
/* Layout */
.cell {
  flex-direction: column;
}

.row {
  display: flex;
}

.status {
  flex-grow: 1;
  line-height: 1.75rem;
  padding: 0 0.33rem;
  text-align: left;
}

.spinner {
  align-self: center;
}

.right {
  display: flex;
  justify-content: center;
  width: 1.5rem;
}

.sheet-active {
  width: 10px;
  height: 10px;

  border-radius: 3.5px;
  align-self: center;
}

.sheet-recent {
  font-size: 9px;
  font-weight: 500;
  line-height: 8px;

  border-radius: 3.5px;
  align-self: center;
}

.hint {
  padding: 0.2rem;
}

/* Theming */
.cell {
  font-size: 0.875rem;
}

.answer {
  font-family: "IBM Plex Mono", "Noto Color Emoji", monospace;
  font-weight: 600;
  text-transform: uppercase;
}

.status {
  margin: 0 1px;
  outline: none;
}

.highlight {
  /* https://stackoverflow.com/a/64127605 */
  margin: 0 -0.4em;
  padding: 0.1em 1.0em 0.1em 0.8em;
  border-radius: 0.75em 0.3em;
  background-image: linear-gradient(90deg,
      oklch(85% 0.10 var(--round-hue) / 10%),
      oklch(91% 0.10 var(--round-hue) / 70%) 4%,
      oklch(91% 0.15 var(--round-hue) / 30%) 92%,
      oklch(91% 0.10 var(--round-hue) / 0%));
}

:global(.filter .highlight:not(.meta)) {
  background-image: linear-gradient(90deg,
      oklch(85% 0 0deg / 10%),
      oklch(91% 0 0deg / 70%) 4%,
      oklch(91% 0 0deg / 30%) 92%,
      oklch(91% 0 0deg / 0%));
}

.solve {
  height: 28.33px;
  line-height: 28px;
  filter: opacity(90%);
  border-radius: 0;
}

.solve:focus-visible {
  /* make Chrome use square outline */
  outline: 2px solid black;
}

.solve:hover {
  filter: none;
}

.solve:hover span {
  transform: scale(110%);
  filter: drop-shadow(0 1px 1px oklch(85% 0 0deg));
}

.sheet-active {
  background-color: oklch(70% 0.17 150);
  border-bottom: 1.5px solid oklch(60% 0.25 150);
}

.sheet-recent {
  font-variant-numeric: tabular-nums;
  font-feature-settings: 'ss01', 'zero';

  padding: 3px 3px 2px;
  border: 1.5px solid oklch(66% 0.17 150);
  color: oklch(60% 0.17 150);
}

.hint {
  font-size: 0.75rem;
  text-align: right;
  color: oklch(55% 0 0deg);
}
</style>
