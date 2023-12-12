<script setup lang="ts">
const props = defineProps<{ puzzle: Puzzle; tabindex: number; }>();
</script>

<template>
  <PuzzleCell :puzzle="puzzle" field="answer" :tabindex="tabindex" v-if="puzzle.answer" />
  <div class="cell status" v-if="!puzzle.answer">
    <span class="highlight">{{ (puzzle.status) ? '✍️' : '' }}
      {{ puzzle.status || 'Not Started' }}</span>
  </div>
</template>

<style scoped>
/* Theming */
.status {
  cursor: pointer;
  box-sizing: border-box;
  border: 1px solid transparent;
  border-radius: 1px;

  line-height: 2em;
  font-size: 0.87rem;
}
</style>
