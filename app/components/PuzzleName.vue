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
  <div class="cell">
    <EditableSpan :value="puzzle.name" readonly :tabindex="tabIndex(focused, 3)"
      :class="puzzle.meta && 'meta'" />
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
  width: 0;
  padding: 0;

  font-size: 0.8rem;
  line-height: 1.75rem;
  align-self: flex-start;
  color: oklch(60% 0.15 245deg);
}

.cell:hover button,
button:hover,
button:focus-visible {
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
