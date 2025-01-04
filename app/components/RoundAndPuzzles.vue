<script setup lang="ts">
const props = defineProps<{
  round: AnnotatedRound;
  i: number;
  focused: FocusInfo;
  observer: IntersectionObserver | undefined;
}>();
const emit = defineEmits<{
  (e: "edit", kind: "puzzle" | "round", id: number): void;
}>();
const store = usePuzzles();
const toast = useToast();

const timelineFromID = (id: number) => `--round-${id}`;
const nextTimelineFromID = (id: number): string | undefined =>
  store.rounds[id + 1] ? timelineFromID(id + 1) : undefined;

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
  if (navigator.clipboard.write) {
    await navigator.clipboard.write([
      new ClipboardItem({
        "text/plain": new Blob([text], { type: "text/plain" }),
        "text/html": new Blob([html], { type: "text/html" }),
      }),
    ]);
  } else {
    await navigator.clipboard.writeText(text);
  }
  toast.add({
    title: "Copied",
    color: "green",
    icon: "i-heroicons-clipboard-document-check",
  });
};
</script>

<template>
  <RoundHeader :round="round" :timeline="timelineFromID(i)"
    :next-timeline="nextTimelineFromID(i)" :observer="observer"
    @copy="() => copy(round.id)" @edit="() => emit('edit', 'round', round.id)" />
  <Puzzle v-for="puzzle in store.puzzlesByRound.get(round.id)" :puzzle="puzzle"
    :round="round" :focused="focused" @edit="() => emit('edit', 'puzzle', puzzle.id)" />
  <div class="empty" v-if="!round.total">
    🫙&hairsp; No Puzzles
  </div>
  <hr>
</template>

<style scoped>
/* Layout */
hr {
  grid-column: 1 / 6;

  margin: 0.5rem -0.5vw 0.5rem -2vw;
  border-bottom: 1px solid oklch(90% 0.03 275deg);
}

.empty {
  grid-column: 1 / 3;
  margin: 0 1.5rem;
}

/* Theming */
.empty {
  font-size: 0.9rem;
  opacity: 50%;
  user-select: none;
}
</style>
