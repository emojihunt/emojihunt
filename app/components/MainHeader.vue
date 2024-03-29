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
    <nav v-if="store.puzzleCount >= 42" @keydown="keydown" class="stop">
      <EmojiNav v-for="round of rounds" :round="round" :observer-fixup="observerFixup"
        :selected="round.id === rounds[focused.index].id" />
    </nav>
    <UTooltip class="ably" :text="connected ? 'Online' : 'Offline'" :open-delay="250"
      :popper="{ placement: 'left', offsetDistance: 0 }">
      <div class="dot" :class="connected && 'connected'"></div>
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
}

nav {
  position: absolute;
  top: 0.75rem;
  right: 1rem;

  display: flex;
  gap: 0.4rem;
}

.ably {
  position: absolute;
  top: 4.5rem;
  right: 0;
  padding: 0.5rem 1.25rem 0.5rem 0.5rem;
}

.dot {
  width: 1rem;
  height: 0.33rem;
  border-radius: 0.25rem;
}

/* Theming */
header {
  background-color: oklch(30% 0.03 275deg);
  border-bottom: 1px solid oklch(70% 0.03 275deg);
  filter: drop-shadow(0 1.5rem 1rem oklch(100% 0 0deg));
  user-select: none;
}

.dot {
  background-color: oklch(62.5% 0.15 30deg);
  filter: drop-shadow(0 0 5px oklch(75% 0.15 30deg));
}

.dot.connected {
  background-color: oklch(55% 0.08 150deg);
  filter: drop-shadow(0 0 3px oklch(65% 0.10 150deg / 50%));
}
</style>
