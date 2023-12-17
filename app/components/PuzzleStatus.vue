<script setup lang="ts">
import { Status } from "../utils/types";

const props = defineProps<{ puzzle: Puzzle; tabindex: number; }>();
const store = usePuzzles();
const open = ref(false);
const saving = ref(false);
const answering = ref<Status | null>(null);
const entry = ref<HTMLDivElement>();

const onStatusSelect = (status: Status) => {
  open.value = false;
  answering.value = null;
  if (!StatusNeedsAnswer(status)) {
    saving.value = true;
    store.updatePuzzle(props.puzzle, { status, answer: "" })
      .finally(() => (saving.value = false));
  } else if (props.puzzle.answer) {
    saving.value = true;
    store.updatePuzzle(props.puzzle, { status })
      .finally(() => (saving.value = false));
  } else {
    answering.value = status;
    nextTick(() => entry.value?.querySelector("span")?.focus());
  }
};
</script>

<template>
  <div class="cell">
    <div v-if="puzzle.answer" class="answer">
      <PuzzleCellInner :puzzle="puzzle" field="answer" :tabindex="tabindex"
        @save="(v) => (saving = v)" />
      <button :title="puzzle.status" :tabindex="tabindex" @click="() => (open = !open)">
        {{ StatusEmoji(puzzle.status) }}
      </button>
      <Spinner v-if="saving" />
    </div>
    <button v-if="!puzzle.answer && !answering" class="status" :tabindex="tabindex"
      @click="() => (open = !open)">
      <span class="highlight">
        {{ StatusEmoji(puzzle.status) }} {{ StatusLabel(puzzle.status) }}
      </span>
      <Spinner v-if="saving" />
    </button>

    <template v-if="answering">
      <div class="answer" ref="entry">
        <PuzzleCellInner :puzzle="puzzle" field="answer" :tabindex="tabindex" editing
          @save="(v) => (saving = v)" />
        <button :title="answering" :tabindex="tabindex"
          @click="() => (answering = null, open = true)">
          {{ StatusEmoji(answering) }}
        </button>
      </div>
      <div class="hint">🎉 Press Enter to record answer</div>
    </template>

    <PuzzleStatusSelector v-if="open" :puzzle="puzzle" @select="onStatusSelect" />
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

.answer button {
  display: flex;
  justify-content: center;
}

.status {
  line-height: 2em;
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
  margin: 0.2rem 0.2rem 0.1rem;
}

/* Theming */
.cell {
  font-size: 0.87rem;
}

.cell:focus-within {
  outline: auto;
}

.answer .inner {
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
