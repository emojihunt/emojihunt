<script setup lang="ts">
const props = defineProps<{ puzzles: any; }>();
const hue = props.puzzles[0].round.color;
</script>

<template>
  <header>
    <div class="cell emoji">{{ puzzles[0].round.emoji }}&#xfe0f;</div>
    <div class="cell round">{{ puzzles[0].round.name }}</div>
    <div class="cell progress">
      {{ puzzles.filter((p: any) => !!p.answer).length }}/{{ puzzles.length }}
    </div>
  </header>
</template>

<style scoped>
/* Layout */
header {
  margin: 2rem 0 0.5rem;
  grid-column: 1 / 6;
  display: grid;
  grid-template-columns: subgrid;
}

div {
  height: 2.8rem;
  line-height: 2.8rem;
}

.round {
  grid-column: 2 / 5;
}

/* Themeing */
header {
  font-size: 1.15rem;
  border: 2px solid transparent;
  border-radius: 2px;
  background: linear-gradient(white, white) padding-box,
    linear-gradient(90deg,
      oklch(77% 0.14 calc(v-bind(hue))),
      oklch(60% 0.21 calc(v-bind(hue) + 75))) border-box;
  filter: drop-shadow(0px 6px 4px oklch(55% 0.21 calc(v-bind(hue)) / 5%));
}

.emoji {
  padding: 0;
  padding-right: 0.5rem;
  font-size: 1.7rem;
  text-align: right;
}

.round {
  color: oklch(25% 0.25 v-bind(hue));
  font-weight: 800;
}

.progress {
  padding-right: 1rem;
  text-align: right;
  font-variant-numeric: diagonal-fractions;
  color: oklch(55% 0.25 calc(v-bind(hue) + 75));
}

.round,
.progress {
  line-height: 3rem;
}
</style>
