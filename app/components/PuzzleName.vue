<script setup lang="ts">
const props = defineProps<{
  puzzle: Puzzle;
  round: AnnotatedRound;
  focused: FocusInfo;
}>();
const emit = defineEmits<{ (e: "edit"): void; }>();
const hue = computed(() => props.round.hue);
</script>

<template>
  <div class="cell" :class="puzzle.meta && 'meta'">
    <EditableSpan :value="puzzle.name" readonly :tabindex="tabIndex(focused, 3)" />
    <button :tabindex="tabIndex(focused, 4)" @click="() => emit('edit')">Edit</button>
  </div>
</template>

<style scoped>
/* Layout */
.cell {
  display: flex;
  overflow: hidden;
}

/* Theming */
.cell {
  font-weight: 430;
  font-size: 0.9rem;
  color: oklch(25% 0.10 275deg);
  border-radius: var(--default-border-radius);
}

button {
  border-radius: 0;
}

.cell:focus-within {
  outline: 2px solid black;
}

.meta:focus-within,
.meta button:focus {
  outline: 2px solid oklch(50% 0.24 v-bind(hue));
}

.meta span {
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

button {
  width: 0;
  padding: 0;

  font-size: 0.8rem;
  line-height: 1.76rem;
  align-self: flex-start;
  color: oklch(60% 0.15 245deg);
}

.meta button {
  color: oklch(60% 0.14 calc(v-bind(hue)))
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
