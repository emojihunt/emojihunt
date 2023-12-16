<script setup lang="ts">
const props = defineProps<{ puzzle: Puzzle; tabindex: number; }>();

const saving = ref(false);
const onSave = (b: boolean) => { saving.value = b; };
</script>

<template>
  <div class="cell" :class="puzzle.answer ? 'solved' : 'unsolved'">
    <PuzzleCellInner :puzzle="puzzle" field="answer" :tabindex="tabindex" @save="onSave"
      v-if="puzzle.answer" />
    <span class="icon" v-if="puzzle.answer" :title="puzzle.status">{{
      puzzle.status == "Solved" ? "🏅" : "🤦‍♀️"
    }}</span>
    <div class="status" v-if="!puzzle.answer">
      <span class="highlight">{{ (puzzle.status) ? '✍️' : '' }}
        {{ puzzle.status || 'Not Started' }}</span>
    </div>
    <Spinner v-if="saving" />
  </div>
</template>

<style scoped>
/* Layout */
.cell {
  display: flex;
  position: relative;
  overflow: hidden;
}

.icon {
  width: 1.75rem;
  line-height: 1.75rem;
}

.status {
  line-height: 2em;
  padding: 0 0.33rem;
  overflow: hidden;
}

.spinner {
  position: absolute;
  right: calc(1.75rem);
  top: calc(1em - 0.5rem);
}

/* Theming */
.solved {
  font-size: 0.87rem;
  font-family: 'IBM Plex Mono', monospace;
  font-weight: 600;
  text-transform: uppercase;
}

.cell:focus-within {
  outline: auto;
}

.icon {
  text-align: center;
  user-select: none;
}

.unsolved {
  cursor: pointer;
}

.status {
  box-sizing: border-box;
  border: 1px solid transparent;
  border-radius: 1px;

  font-size: 0.87rem;
  white-space: nowrap;
  text-overflow: ellipsis;
}
</style>
