<script setup lang="ts">
const props = defineProps<{
  puzzle: Puzzle;
  field: "name" | "location" | "note";
  tabindex: number;
}>();
const store = usePuzzles();
const saving = ref(false);

const save = (updated: string) => {
  saving.value = true;
  store.updatePuzzle(props.puzzle, { [props.field]: updated })
    .finally(() => (saving.value = false));
};
</script>

<template>
  <div class="cell" :class="field">
    <EditableSpan :value="puzzle[field]" :tabindex="tabindex" @save="save" />
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

.location,
.note {
  font-weight: 300;
  font-size: 0.86rem;
}
</style>
