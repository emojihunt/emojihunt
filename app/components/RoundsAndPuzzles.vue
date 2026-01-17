<script setup lang="ts">
const { filter } = defineProps<{ filter: boolean; }>();
const emit = defineEmits<{
  (e: "edit", kind: "puzzle" | "round", id: number): void;
}>();
const { ordering } = usePuzzles();

const focused = ref(3);
const puzz = useTemplateRef("puzz");
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
  if (!puzz.value || !e.target) return;
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
  let i = puzz.value.findIndex((p: any) => p.id === currentID) + delta;
  if (i === undefined) return;
  while (true) {
    if (i < 0 || i >= puzz.value.length) {
      // Focus is in first or last puzzle. Make sure it's visible, then bubble up
      // the event to make the page scroll.
      current?.scrollIntoView();
      return;
    } else if (puzz.value[i]?.isVisible()) {
      puzz.value[i]?.focus();
      e.preventDefault();
      e.stopPropagation();
      return;
    } else {
      i += delta;
    }
  }


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
    (filter ? ordering.value.filter((r) => r.priority) : ordering.value)
      .map((r, i) => [r.id, i])
  )
);

provide(ExpandedKey, ref(0));

defineExpose({
  focus(id: number) {
    puzz.value?.find((p: any) => p.id === id)?.focus();
  },
  navigate() {
    // When using the emoji nav, reset tabindex to puzzle name.
    updateTabIndex(3);
  }
});
</script>

<template>
  <main @keydown="keydown" @focusin="focusin">
    <template v-for="round of ordering">
      <RoundHeader v-if="!filter || round.priority" :key="round.id" :id="round.id"
        :sequence="roundToSequence.get(round.id) || 0" :filter
        @edit="() => emit('edit', 'round', round.id)" />
      <section :class="filter && !round.priority && 'invisible'"
        :style="`--round-hue: ${round.hue}`">
        <Puzzle v-for="puzzle in round.puzzles" ref="puzz" :key="puzzle.id"
          :id="puzzle.id" @edit="() => emit('edit', 'puzzle', puzzle.id)" />
        <div class="empty" v-if="round.total === 0">
          🫙&hairsp; No Puzzles
        </div>
        <hr>
      </section>
    </template>
  </main>
</template>

<style scoped>
/* Layout */
main,
section {
  grid-column: 1 / 6;
  display: grid;
  grid-template-columns: subgrid;
}

.invisible {
  display: none;
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
