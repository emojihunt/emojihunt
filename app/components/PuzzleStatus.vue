<script setup lang="ts">
import { Status } from "../utils/types";

const props = defineProps<{ puzzle: Puzzle; tabindex: number; }>();
const store = usePuzzles();

const input = ref();
const button = ref<HTMLButtonElement>();

const open = ref(false);
const saving = ref(false);
const answering = ref<Status | null>(null);

const select = (status: Status) => {
  open.value = false;
  answering.value = null;
  if (!StatusNeedsAnswer(status)) {
    saving.value = true;
    store.updatePuzzle(props.puzzle, { status, answer: "" })
      .finally(() => (saving.value = false));
    nextTick(() => button.value?.focus());
  } else if (props.puzzle.answer) {
    saving.value = true;
    store.updatePuzzle(props.puzzle, { status })
      .finally(() => (saving.value = false));
  } else {
    answering.value = status;
    nextTick(() => input.value.focus());
  }
};

const save = (answer: string) => {
  if (!answer) return;  // cannot be blank
  answer = answer.toUpperCase();
  if (answering.value) {
    // Answer with state change to "Solved", etc.
    saving.value = true;
    store.updatePuzzle(props.puzzle, { answer, status: answering.value })
      .finally(() => (saving.value = false));
    answering.value = null;
  } else {
    // Regular answer fixup
    saving.value = true;
    store.updatePuzzle(props.puzzle, { answer })
      .finally(() => (saving.value = false));
  }
};

const cancel = () => answering.value && (answering.value = null, open.value = false);
</script>

<template>
  <div class="cell">
    <div v-if="puzzle.answer || answering" class="answer">
      <EditableSpan ref="input" :value="puzzle.answer" :tabindex="tabindex"
        :sticky="!!answering" @save="save" @cancel="cancel" />
      <UTooltip :text="answering || puzzle.status" class="tooltip-lower"
        :popper="{ placement: 'right', offsetDistance: 0 }" :open-delay="500">
        <button :tabindex="tabindex"
          @click="() => answering ? (answering = null, open = true) : (open = !open)">
          {{ StatusEmoji(answering || puzzle.status) }}
        </button>
      </UTooltip>
      <div v-if="answering" class="hint">🎉 Press Enter to record answer</div>
      <Spinner v-if="saving" />
    </div>
    <button v-else ref="button" class="status" :tabindex="tabindex"
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
  display: flex;
  flex-direction: column;

  position: relative;
  overflow: hidden;
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
  font-size: 0.87rem;
}

.cell:focus-within {
  outline: auto;
}

.answer span {
  font-family: 'IBM Plex Mono', monospace;
  font-weight: 600;
  text-transform: uppercase;
}

.answer button {
  line-height: 1.75rem;
  filter: opacity(70%);
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

.hint {
  font-size: 0.76rem;
  text-align: right;
  color: oklch(55% 0 0deg);
}
</style>
