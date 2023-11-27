<script setup lang="ts">
const props = defineProps<{ puzzle: any; }>();
const hue = props.puzzle.round.color;
</script>

<template>
  <span class="row">
    <div class="cell name">
      {{ puzzle.name }}
    </div>
    <div class="cell buttons">
      🌎 ✏️&#xfe0f; 💬
    </div>
    <div class="cell status" v-bind:class="puzzle.answer ? 'solved' : 'unsolved'">
      <span>
        {{ (!puzzle.answer && puzzle.status) ? '✍️' : '' }}
        {{ puzzle.answer || puzzle.status || 'Not Started' }}
      </span>
    </div>
    <div class="cell note">{{ puzzle.location }} {{ puzzle.description || '-' }}</div>
  </span>
</template>

<style scoped>
/* Layout */
.row {
  grid-column: 2 / 6;
  display: grid;
  grid-template-columns: subgrid;
}

.row div {
  height: 1.75em;
  line-height: 1.75rem;
}

/* Themeing */
.row {
  font-size: 0.95rem;
}

.name {
  color: oklch(35% 0.25 v-bind(hue));
  font-weight: 500;
}

.buttons {
  text-align: center;
  letter-spacing: 5px;
  cursor: pointer;
}

.status {
  cursor: pointer;
  box-sizing: border-box;
  border: 1px solid transparent;
  border-radius: 1px;

  font-size: 0.9rem;
}

.solved {
  font-family: 'IBM Plex Mono', monospace;
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

.note {
  font-weight: 300;
  font-size: 0.86rem;
}
</style>
