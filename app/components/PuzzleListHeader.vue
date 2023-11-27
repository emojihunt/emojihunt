<script setup lang="ts">
const props = defineProps<{ puzzles: any; }>();
const hue = props.puzzles[0].round.color;
</script>

<template>
  <header>
    <div class="emoji">{{ puzzles[0].round.emoji }}&#xfe0f;</div>
    <div class="round">{{ puzzles[0].round.name }}</div>
    <div class="progress">
      {{ puzzles.filter((p: any) => !!p.answer).length }}/{{ puzzles.length }}
    </div>
  </header>
</template>

<style scoped>
/* Layout */
header {
  width: 83%;
  grid-column: 1 / 3;
  margin: 2.4rem 0 0.5rem;
  display: flex;

  height: 2.25rem;
  line-height: 2.25rem;
}

/* Themeing */
header {
  font-size: 1.07rem;
  padding: 0 1rem;

  color: oklch(30% 0 0deg);
  background-color: oklch(92% 0.08 v-bind(hue));
  border-radius: 0.5rem;

  box-shadow: 0 0 1px 1px oklch(79% 0.07 v-bind(hue)) inset;
  filter: drop-shadow(0 1px 2px oklch(70% 0.07 v-bind(hue) / 25%));
}

.emoji {
  padding-right: 0.7rem;
}

.round {
  font-weight: 750;
}

.progress {
  flex-grow: 1;
  text-align: right;
  font-variant-numeric: diagonal-fractions;
}
</style>
