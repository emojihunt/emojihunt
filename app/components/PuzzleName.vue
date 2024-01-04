<script setup lang="ts">
const props = defineProps<{
  puzzle: Puzzle;
  round: AnnotatedRound;
  tabindex: number;
}>();
const hue = computed(() => props.round.hue);
</script>

<template>
  <div class="cell" :class="puzzle.meta && 'meta'">
    <EditableSpan :value="puzzle.name" readonly :tabindex="tabindex" />
    <button tabindex="-1">Edit</button>
  </div>
</template>

<style scoped>
/* Layout */
.cell {
  display: flex;
  position: relative;
  overflow: hidden;
}

button {
  position: absolute;
  right: 0;
  height: 1.8rem;
  padding: 0 0.25rem;
}

/* Theming */
.cell {
  font-weight: 430;
  font-size: 0.9rem;
  color: oklch(25% 0.10 275deg);
}

.meta {
  background:
    linear-gradient(68deg,
      oklch(50% 0.24 calc(v-bind(hue))) 0%,
      oklch(50% 0.24 calc(v-bind(hue) + 60)) 20%,
      oklch(50% 0.24 calc(v-bind(hue) + 180)) 100%);
  background-clip: text;
  -webkit-background-clip: text;
  color: transparent;
  font-weight: 550;
}

.cell:focus-within {
  outline: auto black;
}

button {
  font-size: 0.8rem;
  color: oklch(60% 0.15 245deg);
  visibility: hidden;
}

.cell:hover button {
  visibility: visible;
}

button:hover {
  filter: brightness(60%);
}
</style>
