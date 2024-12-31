<script setup lang="ts">
const props = defineProps<{
  rounds: AnnotatedRound[];
  observer: IntersectionObserver | undefined;
  connected: boolean;
}>();
const store = usePuzzles();

// IntersectionObserver doesn't fire with scrollIntoView, so fix up the `stuck`
// classes manually.
const observerFixup = () => {
  if (props.observer) {
    for (const pill of document.querySelectorAll(".ready")) {
      if (pill.getBoundingClientRect().y < 77) {
        pill.classList.add("stuck");
      } else {
        pill.classList.remove("stuck");
      }
    }
  }
};

onMounted(() => document.location.hash && history.pushState(
  "", document.title, window.location.pathname + window.location.search,
));

const [focused, keydown] = useRovingTabIndex(props.rounds.length);
</script>

<template>
  <header>
    <h1>🌊🎨🎡</h1>
    <div class="spacer"></div>
    <nav v-if="store.puzzleCount >= 42" @keydown="keydown" class="stop">
      <EmojiNav v-for="round of rounds" :round="round" :observer-fixup="observerFixup"
        :selected="round.id === rounds[focused.index].id" />
    </nav>
    <UTooltip class="ably" text="Live updates paused. Connecting..." :open-delay="250"
      :popper="{ placement: 'auto-end', offsetDistance: 0 }" v-if="!connected">
      <Icon name="i-heroicons-signal-slash" />
    </UTooltip>
  </header>
</template>

<style scoped>
/* Layout */
header {
  width: 100%;
  height: 6rem;
  position: fixed;
  z-index: 15;

  padding: 0.75rem 1rem 0;

  display: flex;
  align-items: flex-start;
}

h1 {
  min-width: 33%;
}

.spacer {
  flex-grow: 1;
}

nav {
  display: flex;
  gap: 0.4rem;
}

.ably {
  position: absolute;
  bottom: 5px;
  right: 1rem;
  padding: 0 0.5rem;
}

/* Theming */
header {
  color: white;
  background-color: oklch(30% 0.03 275deg);
  border-bottom: 1px solid oklch(70% 0.03 275deg);
  filter: drop-shadow(0 1.5rem 1rem oklch(100% 0 0deg));
  user-select: none;
}

h1 {
  font-size: 1rem;
  letter-spacing: 0.2rem;
  user-select: none;
  opacity: 80%;
  filter: drop-shadow(0 2.5px 4px oklch(82% 0.10 243deg / 20%));
}

/* Animation */
.ably span {
  /* https://stackoverflow.com/a/16344389 */
  animation: blink 1.5s step-start infinite;
  animation-delay: 2.5s;
  opacity: 0;
}
</style>
