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
  </div>
</template>

<style scoped>
/* Layout */
.cell {
  display: flex;
  position: relative;
  overflow: hidden;
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
      oklch(55% 0.28 calc(v-bind(hue) - 30)) 0%,
      oklch(55% 0.28 calc(v-bind(hue) + 60)) 25%,
      oklch(55% 0.28 calc(v-bind(hue) + 150)) 50%);
  background-clip: text;
  -webkit-background-clip: text;
  color: transparent;
  font-weight: 600;
}

.cell:focus-within {
  outline: auto black;
}
</style>
