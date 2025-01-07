<script setup lang="ts">
const props = defineProps<{
  filter: boolean;
  observer: IntersectionObserver | undefined;
}>();
const emit = defineEmits<{
  (e: "edit", kind: "puzzle" | "round", id: number): void;
}>();
const store = usePuzzles();
const toast = useToast();

const focused = reactive({ index: 3 });

const puzzles = useTemplateRef("puzzles");
const keydown = (e: KeyboardEvent) => {
  let delta;
  if (!puzzles.value || !e.target) return;
  else if (e.key === "ArrowUp") delta = -1;
  else if (e.key === "ArrowDown") delta = 1;
  else return;

  // @ts-ignore
  const current = e.target.closest(".puzzle");
  if (!current) {
    // Focus is in round header. Ignore event.
    e.preventDefault();
    e.stopPropagation();
    return;
  }

  const i = puzzles.value.findIndex((p: any) => p.id === current.dataset.puzzle) + delta;
  if (i === undefined || i < 0 || i >= puzzles.value.length) {
    // Focus is in first or last puzzle. Pass through event to scroll page.
    return;
  }
  puzzles.value[i]?.focus();
  e.preventDefault();
  e.stopPropagation();
};

const copy = async (id: number): Promise<void> => {
  const puzzles = store.puzzlesByRound.get(id);
  if (!puzzles) {
    toast.add({
      title: "No puzzles to copy",
      color: "red",
      icon: "i-heroicons-exclamation-triangle",
    });
    return;
  };

  const data = puzzles.map((p) =>
    [p.name, p.answer || `${StatusEmoji(p.status)} ${StatusLabel(p.status)}`]);
  const text = data.map(([a, b]) => `${a}, ${b}`).join("\n");
  const html = "<table>\n" + data.map(([a, b]) =>
    `<tr><td>${a}</td><td>${b}</td></tr>`
  ).join("\n") + "</table>";
  await navigator.clipboard.write([
    new ClipboardItem({
      "text/plain": new Blob([text], { type: "text/plain" }),
      "text/html": new Blob([html], { type: "text/html" }),
    }),
  ]);
  toast.add({
    title: "Copied",
    color: "green",
    icon: "i-heroicons-clipboard-document-check",
  });
};

const roundToSequence = computed(() =>
  new Map(
    (props.filter ? store.rounds.filter((r) => !r.complete) : store.rounds).map((r, i) => [r.id, i]))
);
</script>

<template>
  <section @keydown="keydown">
    <template v-for="round of store.rounds">
      <template v-if="!filter || !round.complete">
        <RoundHeader :round="round" :filter="filter"
          :timeline="timelineFromSequence(roundToSequence.get(round.id)!)"
          :next-timeline="roundToSequence.get(round.id)! < store.rounds.length - 1 ? timelineFromSequence(roundToSequence.get(round.id)! + 1) : undefined"
          :observer="observer" @copy="() => copy(round.id)"
          @edit="() => emit('edit', 'round', round.id)" />
        <template v-for="puzzle in store.puzzlesByRound.get(round.id)">
          <Puzzle v-if="!filter || !puzzle.answer" ref="puzzles" :puzzle="puzzle"
            :round="round" :focused="focused"
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
