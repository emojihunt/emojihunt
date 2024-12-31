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
      <!-- Icon from Heroicons, https://heroicons.com. -->
      <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"
        stroke-width="1.5" stroke="currentColor">
        <path stroke-linecap="round" stroke-linejoin="round"
          d="m3 3 8.735 8.735m0 0a.374.374 0 1 1 .53.53m-.53-.53.53.53m0 0L21 21M14.652 9.348a3.75 3.75 0 0 1 0 5.304m2.121-7.425a6.75 6.75 0 0 1 0 9.546m2.121-11.667c3.808 3.807 3.808 9.98 0 13.788m-9.546-4.242a3.733 3.733 0 0 1-1.06-2.122m-1.061 4.243a6.75 6.75 0 0 1-1.625-6.929m-.496 9.05c-3.068-3.067-3.664-7.67-1.79-11.334M12 12h.008v.008H12V12Z" />
      </svg>
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
  bottom: 4px;
  right: 1rem;
  padding: 0 0.5rem;
}

svg {
  display: inline-block;
  height: 1rem;
  width: 1rem;
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

svg {
  opacity: 0;
}

/* Animation */
svg {
  /* https://stackoverflow.com/a/16344389 */
  animation: blink 1.5s step-start infinite;
  animation-delay: 2.5s;
}
</style>
