<script setup lang="ts">
const props = defineProps<{ puzzle: Puzzle; round: Round; }>();
const store = usePuzzles();

const input = useTemplateRef("input");
const button = useTemplateRef("button");

const open = ref(false);
const saving = ref(false);
const answering = ref<Status | null>(null);

const select = (status: Status) => {
  open.value = false;
  answering.value = null;
  if (!StatusNeedsAnswer(status)) {
    saving.value = true;
    store.updatePuzzleOptimistic(props.puzzle.id, { status, answer: "" })
      .finally(() => (saving.value = false));
    nextTick(() => button.value?.focus());
  } else if (props.puzzle.answer) {
    saving.value = true;
    store.updatePuzzleOptimistic(props.puzzle.id, { status })
      .finally(() => (saving.value = false));
  } else {
    answering.value = status;
    nextTick(() => input.value?.focus());
  }
};

const save = (answer: string) => {
  if (!answer) return;  // cannot be blank
  answer = answer.toUpperCase();
  if (answering.value) {
    // Answer with state change to "Solved", etc.
    saving.value = true;
    store.updatePuzzleOptimistic(props.puzzle.id, { answer, status: answering.value, voice_room: "" })
      .finally(() => (saving.value = false));
    answering.value = null;
  } else {
    // Regular answer fixup
    saving.value = true;
    store.updatePuzzleOptimistic(props.puzzle.id, { answer })
      .finally(() => (saving.value = false));
  }
};

const hue = computed(() => props.round.hue);
const cancel = () => answering.value && (answering.value = null, open.value = false);
</script>

<template>
  <div class="cell">
    <div v-if="puzzle.answer || answering" class="answer">
      <EditableSpan ref="input" :value="puzzle.answer" :tabsequence="5"
        :sticky="!!answering" @save="save" @cancel="cancel" />
      <ETooltip :text="answering || puzzle.status" placement="left">
        <button :data-tabsequence="6"
          @click="() => answering ? (answering = null, open = true) : (open = !open)">
          {{ StatusEmoji(answering || puzzle.status) }}
        </button>
      </ETooltip>
      <div v-if="answering" class="hint">🎉 Press Enter to record answer</div>
      <Spinner v-if="saving" />
    </div>
    <button v-else ref="button" class="status" :data-tabsequence="56"
      @click="() => (open = !open)">
      <span class="highlight">
        {{ StatusEmoji(puzzle.status) }} {{ StatusLabel(puzzle.status) }}
      </span>
      <Spinner v-if="saving" />
    </button>
    <PuzzleStatusSelector v-if="open" :puzzle="puzzle" @select="select" />
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

.status {
  line-height: 1.8rem;
  padding: 0 0.33rem;
  text-align: left;
}

.spinner {
  position: absolute;
  top: calc(1em - 0.5rem);
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
  line-height: 1.75rem;
  filter: opacity(90%);
}

.answer button:hover {
  transform: scale(110%);
  filter: drop-shadow(0 1px 1px oklch(85% 0 0deg));
  /* also clears prior opacity() filter */
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
      oklch(85% 0.10 v-bind(hue) / 10%),
      oklch(91% 0.10 v-bind(hue) / 70%) 4%,
      oklch(91% 0.15 v-bind(hue) / 30%) 92%,
      oklch(91% 0.10 v-bind(hue) / 0%));
}

.hint {
  font-size: 0.75rem;
  text-align: right;
  color: oklch(55% 0 0deg);
}
</style>
