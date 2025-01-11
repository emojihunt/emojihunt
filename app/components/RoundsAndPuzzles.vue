<script setup lang="ts">
const { filter, observer } = defineProps<{
  filter: boolean;
  observer: IntersectionObserver | undefined;
}>();
const emit = defineEmits<{
  (e: "edit", kind: "puzzle" | "round", id: number): void;
}>();
const store = usePuzzles();

const focused = ref(3);
const puzzles = useTemplateRef("puzzles");
const updateTabIndex = (i: number) => {
  focused.value = i;
  document.querySelectorAll("[data-tabsequence]").forEach(
    (el) => el.setAttribute("tabIndex", "-1")
  );
  document.querySelectorAll(`[data-tabsequence="${i}"]`).forEach(
    (el) => el.setAttribute("tabIndex", "0"));
  if (i === 5 || i === 6) {
    document.querySelectorAll(`[data-tabsequence="56"]`).forEach(
      (el) => el.setAttribute("tabIndex", "0"));
  }
};
const keydown = (e: KeyboardEvent) => {
  let delta = 0;
  if (!puzzles.value || !e.target) return;
  else if (e.key === "ArrowUp") delta = -1;
  else if (e.key === "ArrowDown") delta = 1;
  else if (e.key === "ArrowRight") {
    if (focused.value < 8) {
      updateTabIndex(focused.value + 1);
    }
  } else if (e.key === "ArrowLeft") {
    if (focused.value > 0) {
      updateTabIndex(focused.value - 1);
    }
  } else return;

  if (!(e.target instanceof HTMLElement)) return;
  const current = e.target.closest<HTMLElement>(".puzzle");
  if (!current) {
    // Focus is in round header. Ignore event.
    e.preventDefault();
    e.stopPropagation();
    return;
  }
  const currentID = parseInt(current.dataset.puzzle!);

  // Note: we assume (unsafely?) that the `puzzles` array matches the order of
  // puzzles on the page.
  const i = puzzles.value.findIndex((p: any) => p.id === currentID) + delta;
  if (i === undefined || i < 0 || i >= puzzles.value.length) {
    // Focus is in first or last puzzle. Make sure it's visible, then bubble up
    // the event to make the page scroll.
    current?.scrollIntoView();
    return;
  }

  puzzles.value[i]?.focus();
  e.preventDefault();
  e.stopPropagation();
};
const focusin = (e: FocusEvent) => {
  if (!(e.target instanceof HTMLElement)) return;
  const cell = e.target.closest<HTMLElement>("[data-tabsequence]");
  const index = cell?.dataset.tabsequence;
  if (!index) return;
  let target;
  if (index === "56") {
    if (focused.value === 5 || focused.value === 6) return;
    target = 5;
  } else {
    if (parseInt(index) === focused.value) return;
    target = parseInt(index);
  }
  updateTabIndex(target);
  e.preventDefault();
  e.stopPropagation();
};
onMounted(() => updateTabIndex);

const roundToSequence = computed(() =>
  new Map(
    (filter ? store.rounds.filter((r) => !r.complete) : store.rounds).map((r, i) => [r.id, i]))
);

defineExpose({
  focus(id: number) {
    puzzles.value?.find((p: any) => p.id === id)?.focus();
  },
  navigate() {
    // When using the emoji nav, reset tabindex to puzzle name.
    updateTabIndex(3);
  }
});
</script>

<template>
  <section @keydown="keydown" @focusin="focusin">
    <template v-for="round of store.rounds">
      <RoundHeader v-if="!filter || !round.complete" :round="round"
        :first="roundToSequence.get(round.id) === 0" :filter="filter"
        :timeline="timelineFromSequence(roundToSequence.get(round.id)!)"
        :next-timeline="roundToSequence.get(round.id)! < store.rounds.length - 1 ? timelineFromSequence(roundToSequence.get(round.id)! + 1) : undefined"
        :observer="observer" @edit="() => emit('edit', 'round', round.id)"
        :key="round.id" />
      <Puzzle v-for="puzzle in store.puzzlesByRound.get(round.id)" :key="puzzle.id"
        ref="puzzles" :puzzle="puzzle" :round="round"
        @edit="() => emit('edit', 'puzzle', puzzle.id)" />
      <div class="empty" v-if="(!filter || !round.complete) && !round.total">
        🫙&hairsp; No Puzzles
      </div>
      <hr v-if="!filter || !round.complete">
    </template>
  </section>
</template>

<style scoped>
/* Layout */
section {
  grid-column: 1 / 6;
  display: grid;
  grid-template-columns: subgrid;
}

hr {
  grid-column: 1 / 6;

  margin: 0.5rem -0.5vw 0.5rem calc(-1 * var(--nav-margin));
  border-bottom: 1px solid oklch(90% 0.03 275deg);
}

.empty {
  grid-column: 1 / 3;
  margin: 0 1.5rem;
}

/* Theming */
.empty {
  font-size: 0.875rem;
  opacity: 50%;
  user-select: none;
}
</style>
