<script setup lang="ts">
const props = defineProps<{
  puzzle: Puzzle;
  field: "location" | "note";
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
    <UTooltip v-if="false" text="Placeholder" :open-delay="500"
      :popper="{ placement: 'right', offsetDistance: 0 }">
      <span class="emoji">⏰</span>
    </UTooltip>
    <EditableSpan :value="puzzle[field]" :tabindex="tabindex" @save="save" />
    <Spinner v-if="saving" class="spinner" />
  </div>
</template>

<style scoped>
/* Layout */
.cell {
  display: flex;
  position: relative;
  overflow: hidden;
}

.emoji {
  width: 1.5rem;
  text-align: center;
}

.spinner {
  right: 0.33rem;
  top: calc((1.8rem - 1em) / 2);
}

/* Theming */
.cell {
  font-weight: 300;
  font-size: 0.86rem;
}

.cell:focus-within {
  outline: auto;
}

.emoji {
  line-height: 1.75rem;
  filter: opacity(70%);
  user-select: none;
}

.emoji:hover {
  transform: scale(110%);
  filter: drop-shadow(0 1px 1px oklch(85% 0 0deg));
}
</style>
