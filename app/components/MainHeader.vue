<script setup lang="ts">
const props = defineProps<{
  rounds: RoundStats[];
  observer: IntersectionObserver | undefined;
}>();

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
    <nav v-if="rounds.length > 2" @keydown="keydown" class="stop">
      <EmojiNav v-for="round of rounds" :round="round" :observer-fixup="observerFixup"
        :selected="round.id == rounds[focused.index].id" />
    </nav>
  </header>
</template>

<style scoped>
/* Layout */
header {
  width: 100%;
  height: 6rem;
  position: fixed;
  z-index: 10;
}

nav {
  position: absolute;
  top: 0.75rem;
  right: 1.5rem;

  display: flex;
  gap: 0.4rem;
}

/* Themeing */
header {
  background-color: oklch(30% 0.03 275deg);
  border-bottom: 1px solid oklch(50% 0.03 275deg);
  filter: drop-shadow(0 1.5rem 1rem oklch(100% 0 0deg));

  user-select: none;
}
</style>
