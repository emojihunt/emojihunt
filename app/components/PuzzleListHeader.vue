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
}

.emoji {
  padding: 0;
  padding-right: 0.5rem;
  font-size: 1.7rem;
  text-align: right;
}

.round {
  font-weight: 800;
}

.progress {
  padding-right: 1rem;
  text-align: right;
  font-variant-numeric: diagonal-fractions;
}

.round,
.progress {
  line-height: 3rem;
}
</style>
