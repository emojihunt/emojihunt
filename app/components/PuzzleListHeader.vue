<script setup lang="ts">
const props = defineProps<{ puzzles: any; }>();
const hue = props.puzzles[0].round.color;
</script>

<template>
  <header class="pill">
    <div class="emoji">{{ puzzles[0].round.emoji }}&#xfe0f;</div>
    <div class="round">{{ puzzles[0].round.name }}</div>
    <div class="progress">
      {{ puzzles.filter((p: any) => !!p.answer).length }}/{{ puzzles.length }}
    </div>
  </header>
  <header class="titles">
    <div class="cell">Status &bull; Answer</div>
    <div class="cell">Location</div>
    <div class="cell">Note</div>
  </header>
</template>

<style scoped>
/* Layout */
.pill {
  grid-column: 1 / 3;
  width: 83%;
  margin: 2.4rem 0 0.5rem;
  display: flex;

  height: 2.25rem;
  line-height: 2.35rem;
}

.titles {
  grid-column: 3 / 6;
  display: grid;
  grid-template-columns: subgrid;
  align-self: flex-end;
}

/* Themeing */
.pill {
  font-size: 1.07rem;
  padding: 0 1.2rem;
  gap: 0.6rem;

  color: oklch(48% 0.075 v-bind(hue));
  border-radius: 0.6rem;
  border: 1.5px solid transparent;
  background: linear-gradient(68deg, oklch(100% 0 0deg / 94%),
      oklch(100% 0 0deg / 87%)) padding-box,
    linear-gradient(68deg,
      oklch(77% 0.10 v-bind(hue)) 7%,
      oklch(60% 0.30 calc(v-bind(hue) + 60))) border-box;
  filter: drop-shadow(0 1px 2px oklch(70% 0.07 v-bind(hue) / 25%));
}

.round {
  font-weight: 715;
}

.progress {
  flex-grow: 1;
  text-align: right;
  font-variant-numeric: diagonal-fractions;
  color: oklch(60% 0.30 calc(v-bind(hue) + 60));
  user-select: none;
}

.titles {
  padding-bottom: 0.2rem;

  font-size: 0.8rem;
  font-weight: 430;
  color: oklch(55% 0 0deg);

  user-select: none;
}
</style>
