<script setup lang="ts">
const { id } = defineProps<{ id: number; }>();

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
      .finally(() => (saving.value = false));
    answering.value = null;
  } else {
    // Regular answer fixup
    saving.value = true;
    updatePuzzleOptimistic(id, { answer })
      .finally(() => (saving.value = false));
  }
};
const cancel = () => answering.value && (answering.value = null, expanded && (expanded.value = 0));

const items = computed(() =>
  Object.values(Status).filter((s) => s !== puzzle.status).map(
    (s) => ({ id: s, emoji: StatusEmoji(s), name: StatusLabel(s) })
  )
);
const select = (status: Status) => {
  if (expanded) expanded.value = 0;
  answering.value = null;
  if (!StatusNeedsAnswer(status)) {
    saving.value = true;
    updatePuzzleOptimistic(id, { status, answer: "" })
      .finally(() => (saving.value = false));
    nextTick(() => button.value?.focus());
  } else if (puzzle.answer) {
    saving.value = true;
    updatePuzzleOptimistic(id, { status })
      .finally(() => (saving.value = false));
  } else {
    answering.value = status;
    nextTick(() => input.value?.focus());
  }
};
</script>

<template>
  <div class="cell">
    <div v-if="puzzle.answer || answering" class="answer">
      <EditableSpan ref="input" :value="puzzle.answer" :tabsequence="5"
        :sticky="!!answering" @save="save" @cancel="cancel" />
      <ETooltip :text="answering || puzzle.status" side="left">
        <button :data-tabsequence="6"
          @click="() => answering ? (answering = null, expanded = id) : expanded = (expanded === id ? 0 : id)">
          <span>{{ StatusEmoji(answering || puzzle.status) }}</span>
        </button>
      </ETooltip>
      <div v-if="answering" class="hint">🎉 Press Enter to record answer</div>
      <Spinner v-if="saving" />
    </div>
    <button v-else ref="button" class="status" :data-tabsequence="56"
      @click="() => expanded = (expanded === id ? 0 : id)">
      <span class="highlight" :class="puzzle.meta && 'meta'">
        {{ StatusEmoji(puzzle.status) }} {{ StatusLabel(puzzle.status) }}
      </span>
      <Spinner v-if="saving" />
    </button>
    <OptionPane v-if="expanded === id" :items="items" @select="select" />
  </div>
</template>

<style scoped>
/* Layout */
.cell {
  flex-direction: column;
}

.answer {
  display: grid;
  grid-template-columns: 5fr 1.5rem;
}

.answer div {
  display: flex;
}

.answer button {
  flex-grow: 1;
  text-align: center;
  align-self: flex-start;
}

.answer button span {
  display: inline-block;
}

.status {
  line-height: 1.75rem;
  padding: 0 0.33rem;
  text-align: left;
}

.spinner {
  position: absolute;
  top: 5px;
  right: 0.4rem;
}

.answer .spinner {
  right: 1.75rem;
}

.hint {
  grid-column: 1 / 3;
  padding: 0.2rem 0.2rem 0.1rem;
}

/* Theming */
.cell {
  font-size: 0.875rem;
}

.answer span {
  font-family: "IBM Plex Mono", "Noto Color Emoji", monospace;
  font-weight: 600;
  text-transform: uppercase;
}

.answer button {
  line-height: calc(1.75rem - 1px);
  filter: opacity(90%);
  border-radius: 0;
}

.answer button:hover {
  filter: none;
}

.answer button:hover span {
  transform: scale(110%);
  filter: drop-shadow(0 1px 1px oklch(85% 0 0deg));
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

.hint {
  font-size: 0.75rem;
  text-align: right;
  color: oklch(55% 0 0deg);
}
</style>
