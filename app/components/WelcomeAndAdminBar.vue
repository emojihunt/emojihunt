<script setup lang="ts">
const emit = defineEmits<{
  (e: "click", kind: "round" | "puzzle" | "admin"): void;
}>();
const store = usePuzzles();
const config = useAppConfig();

const display = ref<string | undefined>();
const recalculate = () => {
  display.value = undefined;
  if (!store.nextHunt) return;

  let delta = store.nextHunt.getTime() - new Date().getTime();
  delta = Math.floor(delta / 1000);
  if (delta <= 0) return;

  const seconds = delta % 60;
  display.value = `${seconds} second${seconds === 1 ? '' : 's'}!!!!`;
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

  display.value = `${delta} day${delta === 1 ? '' : 's'}, ${hours} hour${hours === 1 ? '' : 's'}...`;
};

recalculate();
onMounted(() => {
  recalculate();
  if (display.value && !store.rounds.length) {
    setInterval(recalculate, 200);
  }
});

const round = ref();
const puzzle = ref();
const admin = ref();
const focus = (kind: "round" | "puzzle" | "admin") => nextTick(() => {
  if (kind === "round") round.value?.focus();
  else if (kind === "puzzle") puzzle.value?.focus();
  else admin.value?.focus();
});
defineExpose({ focus });
</script>

<template>
  <section v-if="!store.rounds.length">
    <NuxtLink :to="display ? 'https://www.isithuntyet.info' : config.huntURL">
      <template v-if="display">
        ⏳&hairsp; <span>Hunt begins in {{ display }}</span>
      </template>
      <template v-else>
        🎉&nbsp; <span class="hunt">HUNT HUNT HUNT!!!!!</span>
      </template>
    </NuxtLink>
    <hr>
  </section>
  <footer>
    <fieldset>
      <button ref="round" @click="() => emit('click', 'round')"
        :disabled="!!display && !store.rounds.length">○
        Add
        Round</button>
      <button ref="puzzle" @click="() => emit('click', 'puzzle')"
        :disabled="!store.rounds.length">▢ Add Puzzle</button>
      <button ref="admin" @click="emit('click', 'admin')">◆
        Admin</button>
    </fieldset>
  </footer>
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

  margin: 0.5rem -0.5vw 0.5rem calc(-1 * var(--nav-margin));
  border-bottom: 1px solid oklch(90% 0.03 275deg);
}

footer {
  grid-column: 1 / 6;
  display: flex;
  flex-direction: column;
  align-items: flex-end;

  gap: 0.4rem;
  margin: 0 0.75rem;
  padding-bottom: 18vh;
}

button {
  padding: 0.25rem;
  margin: 0 0.25rem;
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
  color: oklch(40% 0.15 245deg);
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
  color: oklch(40% 0.15 245deg);
}

button[disabled] {
  filter: grayscale(100%) opacity(80%);
  pointer-events: none;
}

button:focus-visible,
a:focus-visible {
  outline-color: oklch(50% 0.15 245deg) !important;
}
</style>
