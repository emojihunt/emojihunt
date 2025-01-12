<script setup lang="ts">
const { id } = defineProps<{ id: number; }>();
const emit = defineEmits<{ (e: "edit"): void; }>();

const { puzzles } = usePuzzles();
const puzzle = puzzles.get(id)!;
</script>

<template>
  <div class="cell" :class="puzzle.meta && 'meta'">
    <EditableSpan :value="puzzle.name" readonly :tabsequence="3" />
    <button :data-tabsequence="4" @click="() => emit('edit')">Edit</button>
  </div>
</template>

<style scoped>
/* Theming */
.cell {
  font-weight: 430;
  font-size: 0.875rem;
  color: oklch(25% 0 0deg);
}

button {
  border-radius: 0;
}

.meta:focus-within,
.meta button:focus-visible {
  outline-color: oklch(50% 0.24 var(--round-hue)) !important;
}

.meta span {
  background:
    linear-gradient(68deg,
      oklch(50% 0.24 var(--round-hue)) 0%,
      oklch(50% 0.24 calc(var(--round-hue) + 60)) 20%,
      oklch(50% 0.24 calc(var(--round-hue) + 180)) 100%);
  background-clip: text;
  -webkit-background-clip: text;
  color: transparent;
  font-weight: 550;
}

button {
  width: 0;
  padding: 0;

  font-size: 0.8125rem;
  line-height: 28px;
  align-self: flex-start;
  color: oklch(60% 0.15 245deg);
}

.meta button {
  color: oklch(60% 0.14 var(--round-hue));
}

.cell:hover button,
button:hover,
button:focus {
  width: auto;
  padding: 0 0.33rem;
}

.cell:hover span {
  white-space: unset;
}

button:hover {
  color: oklch(40% 0.15 245deg);
}
</style>
