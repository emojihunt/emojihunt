<script setup lang="ts">
const props = defineProps<{ puzzle: Puzzle; tabindex: number; }>();
</script>

<template>
  <PuzzleCell :puzzle="puzzle" field="answer" :tabindex="tabindex" v-if="puzzle.answer" />
  <div class="status" v-if="!puzzle.answer">
    <span class="highlight">{{ (puzzle.status) ? '✍️' : '' }}
      {{ puzzle.status || 'Not Started' }}</span>
  </div>
</template>

<style scoped>
/* Layout */
.status {
  line-height: 2em;
  padding: 0 0.33rem;
  overflow: hidden;
}

/* Theming */
.status {
  cursor: pointer;
  box-sizing: border-box;
  border: 1px solid transparent;
  border-radius: 1px;

  font-size: 0.87rem;
  white-space: nowrap;
  text-overflow: ellipsis;
}
</style>
