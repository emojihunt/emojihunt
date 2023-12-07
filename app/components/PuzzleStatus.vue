<script setup lang="ts">
const props = defineProps<{ puzzle: Puzzle; }>();
</script>

<template>
  <div class="cell status" v-bind:class="puzzle.answer ? 'solved' : 'unsolved'">
    <span>
      {{ (!puzzle.answer && puzzle.status) ? '✍️' : '' }}
      {{ puzzle.answer || puzzle.status || 'Not Started' }}
    </span>
  </div>
</template>

<style scoped>
/* Theming */
div {
  font-size: 0.9rem;
  line-height: 2em;
}

div:hover {
  white-space: unset;
}

.status {
  cursor: pointer;
  box-sizing: border-box;
  border: 1px solid transparent;
  border-radius: 1px;

  font-size: 0.87rem;
}

.solved {
  font-family: 'IBM Plex Mono', monospace;
  font-weight: 600;
}

.unsolved span {
  /* https://stackoverflow.com/a/64127605 */
  margin: 0 -0.4em;
  padding: 0.1em 1.0em 0.1em 0.8em;
  border-radius: 0.75em 0.3em;
  background-image: linear-gradient(90deg,
      oklch(85% 0.25 100deg / 10%),
      oklch(91% 0.19 100deg / 70%) 4%,
      oklch(91% 0.25 100deg / 30%) 92%,
      oklch(91% 0.25 100deg / 0%));
}
</style>
