<script setup lang="ts">
const store = usePuzzles();
const config = useAppConfig();

const display = ref<string | undefined>();
const recalculate = () => {
  console.warn("Recalculating...");
  display.value = undefined;
  if (!store.next_hunt) return;

  let delta = store.next_hunt.getTime() - new Date().getTime();
  delta = Math.floor(delta / 1000);
  if (delta <= 0) return;

  const seconds = delta % 60;
  display.value = `${seconds} second${seconds > 1 ? 's' : ''}!!!!`;
  delta = Math.floor(delta / 60);
  if (delta <= 0) return;

  const minutes = delta % 60;
  display.value = `${minutes}:${String(seconds).padStart(2, "0")}!!!`;
  delta = Math.floor(delta / 60);
  if (delta <= 0) return;

  const hours = delta % 24;
  display.value = `${hours}:${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}!!`;
  delta = Math.floor(delta / 24);
  if (delta <= 0) return;

  display.value = `${delta} day${delta > 1 ? 's' : ''}...`;
};
defineExpose({ hunt: !display });

recalculate();
onMounted(() => {
  recalculate();
  if (display.value && !store.rounds.length) {
    setInterval(recalculate, 200);
  }
});

const modal = ref<"round" | "puzzle" | null>(null);
const round = ref();
const puzzle = ref();
const close = () => {
  if (modal.value === "round") nextTick(() => round.value?.focus());
  else nextTick(() => puzzle.value?.focus());
  modal.value = null;
};
</script>

<template>
  <section v-if="!store.rounds.length">
    <NuxtLink :href="display ? 'https://www.isithuntyet.info' : config.huntURL">
      <template v-if="display">
        ⏳&hairsp; <span>Hunt begins in {{ display }}</span>
      </template>
      <template v-else>
        🎉&nbsp; <span class="hunt">HUNT HUNT HUNT!!!!!</span>
      </template>
    </NuxtLink>
    <hr>
  </section>
  <fieldset>
    <button ref="round" @click="() => (modal = 'round')"
      :disabled="!!display && !store.rounds.length">○ Add Round</button>
    <button ref="puzzle" @click="() => (modal = 'puzzle')"
      :disabled="!!display && !store.rounds.length">▢ Add Puzzle</button>
    <button>◆ Admin</button>
  </fieldset>
  <AddRoundPuzzleModal :open="!!modal" :kind="modal" @close="close" />
</template>

<style scoped>
/* Layout */
section {
  grid-column: 1 / 6;
  margin: 2rem 0 0 0;
  z-index: 20;

  display: grid;
  grid-template-columns: subgrid;
}

a {
  grid-column: 1 / 3;
  padding: 0.5rem 0.25rem;
}

hr {
  grid-column: 1 / 6;

  margin: 0.5rem -0.5vw 0.5rem -2vw;
  border-bottom: 1px solid oklch(90% 0.03 275deg);
}

fieldset {
  grid-column: 5;
  display: flex;
  gap: 0.5rem;

  justify-content: flex-end;
  margin: 0 1rem;
}

button {
  padding: 0.25rem;
}

/* Theming */
a {
  font-weight: 500;
  font-size: 0.95rem;
  color: oklch(60% 0.15 245deg);
  text-decoration: none;
}

span {
  font-feature-settings: 'tnum';
}

a:hover span {
  filter: brightness(60%);
}

.hunt {
  font-weight: bold;
}

button {
  font-weight: 350;
  font-size: 0.85rem;
  color: oklch(60% 0.15 245deg);
}

button:hover {
  filter: brightness(60%);
}

button[disabled] {
  filter: grayscale(100%) opacity(80%);
  pointer-events: none;
}

a,
button {
  outline-color: oklch(40% 0.15 245deg);
}
</style>
