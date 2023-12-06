<script setup lang="ts">
const props = defineProps<{
  rounds: RoundStats[];
  observer: IntersectionObserver | undefined;
}>();

// IntersectionObserver doesn't fire with scrollIntoView, so fix up the `stuck`
// classes manually.
const observerFixup = () => {
  if (props.observer) {
    for (const pill of document.querySelectorAll(".pill")) {
      if (pill.getBoundingClientRect().y < 75) {
        pill.classList.add("stuck");
      } else {
        pill.classList.remove("stuck");
      }
    }
  }
};

// Use the "roving tabindex" technique to avoid having to tab through every
// round in the navbar.
const nav = ref<HTMLElement>();
const selected = reactive({ idx: 0 });
const keydown = (e: KeyboardEvent) => {
  if (e.key == "ArrowRight") {
    if (selected.idx < props.rounds.length - 1) selected.idx += 1;
  } else if (e.key == "ArrowLeft") {
    if (selected.idx > 0) selected.idx -= 1;
  } else {
    return;
  }
  // @ts-ignore
  setTimeout(() => nav.value?.querySelector("[tabindex='0']")?.focus(), 0);
  e.preventDefault();
};

onMounted(() => document.location.hash && history.pushState(
  "", document.title, window.location.pathname + window.location.search,
));
</script>

<template>
  <header>
    <nav v-if="rounds.length > 2" @keydown="keydown" ref="nav">
      <NavbarEmoji v-for="round of rounds" :round="round" :observer-fixup="observerFixup"
        :selected="round.id == rounds[selected.idx].id" />
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
  top: 1rem;
  right: 1.75rem;

  display: flex;
  gap: 0.5rem;
}


/* Themeing */
header {
  background-color: oklch(98% 0 0deg);
  border-bottom: 1px solid oklch(80% 0 0deg);
  filter: drop-shadow(0 1.5rem 1rem oklch(100% 0 0deg));

  user-select: none;
}
</style>
