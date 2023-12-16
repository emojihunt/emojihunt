<script setup lang="ts">
const props = defineProps<{
  puzzle: Puzzle;
  field: "name" | "location" | "description";
  tabindex: number;
  readonly?: boolean;
}>();

const saving = ref(false);
const onSave = (b: boolean) => { saving.value = b; };
</script>

<template>
  <div class="cell" :class="field">
    <PuzzleCellInner :puzzle="puzzle" :field="field" :tabindex="tabindex"
      :readonly="readonly" @save="onSave" />
    <Spinner v-if="saving" />
  </div>
</template>

<style scoped>
/* Layout */
.cell {
  display: flex;
  position: relative;
  overflow: hidden;
}

.spinner {
  position: absolute;
  right: 0.33rem;
  top: calc(1em - 0.5rem);
}

/* Theming */
.cell:focus-within {
  outline: auto;
}

.name {
  font-weight: 430;
  font-size: 0.9rem;
  color: oklch(25% 0.10 275deg);
}

.location,
.description {
  font-weight: 300;
  font-size: 0.86rem;
}
</style>
