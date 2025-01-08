<script setup lang="ts">
const props = defineProps<{
  filter: boolean;
  observer: IntersectionObserver | undefined;
}>();
const emit = defineEmits<{
  (e: "edit", kind: "puzzle" | "round", id: number): void;
}>();
const store = usePuzzles();

const focused = reactive({ index: 3 });
const puzzles = useTemplateRef("puzzles");
const setTabIndex = (i: number, focused: boolean) =>
  document.querySelectorAll(`[data-tabsequence="${i}"]`).forEach(
    (el) => el.setAttribute("tabIndex", focused ? "0" : "-1"));
const keydown = (e: KeyboardEvent) => {
  let delta = 0;
  let previous;
  if (!puzzles.value || !e.target) return;
  else if (e.key === "ArrowUp") delta = -1;
  else if (e.key === "ArrowDown") delta = 1;
  else if (e.key === "ArrowRight") {
    if (focused.index < 8) {
      previous = focused.index;
      focused.index += 1;
    }
  } else if (e.key === "ArrowLeft") {
    if (focused.index > 0) {
      previous = focused.index;
      focused.index -= 1;
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

  if (previous !== undefined) {
    console.warn(previous, focused);
    setTabIndex(previous, false);
    setTabIndex(focused.index, true);
    // Handle combined status/answer element
    if (focused.index === 5 || focused.index === 6) {
      setTabIndex(56, true);
    } else if (previous === 5 || previous === 6) {
      setTabIndex(56, false);
    }
  }

  puzzles.value[i]?.focus();
  e.preventDefault();
  e.stopPropagation();
};
onMounted(() => {
  for (var i = 0; i < 9; i++) {
    setTabIndex(i, i == focused.index);
  }
  setTabIndex(56, false);
});

const roundToSequence = computed(() =>
  new Map(
    (props.filter ? store.rounds.filter((r) => !r.complete) : store.rounds).map((r, i) => [r.id, i]))
);

defineExpose({
  focus(id: number) {
    puzzles.value?.find((p: any) => p.id === id)?.focus();
  },
});
</script>

<template>
  <section @keydown="keydown">
    <template v-for="round of store.rounds" :key="round.id">
      <template v-if="!filter || !round.complete">
        <RoundHeader :round="round" :filter="filter"
          :timeline="timelineFromSequence(roundToSequence.get(round.id)!)"
          :next-timeline="roundToSequence.get(round.id)! < store.rounds.length - 1 ? timelineFromSequence(roundToSequence.get(round.id)! + 1) : undefined"
          :observer="observer" @edit="() => emit('edit', 'round', round.id)" />
        <template v-for="puzzle in store.puzzlesByRound.get(round.id)" :key="puzzle.id">
          <Puzzle v-if="!filter || !puzzle.answer" ref="puzzles" :puzzle="puzzle"
            :round="round" :focused="focused" :filter="filter"
            @edit="() => emit('edit', 'puzzle', puzzle.id)" />
        </template>
        <div class="empty" v-if="!round.total">
          🫙&hairsp; No Puzzles
        </div>
        <hr>
      </template>
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
