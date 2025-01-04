<script setup lang="ts">
const props = defineProps<{
  rounds: AnnotatedRound[];
  observer: IntersectionObserver | undefined;
}>();
const url = useRequestURL();

// Navigate to anchors without changing the fragment
const goto = (round: AnnotatedRound) => {
  const id = (new URL(`#${round.anchor}`, url)).hash; // escaping
  if (round.id == props.rounds[0].id) {
    // Sometimes scrolling to the first anchor doesn't work.
    window.scrollTo({ top: 0 });
  } else {
    document.querySelector(id)?.scrollIntoView();
  }
  document.querySelector<HTMLElement>(`${id} ~ .row [tabIndex='0']`)?.focus();

  // IntersectionObserver doesn't fire with scrollIntoView, so fix up the
  // `stuck` classes manually.
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

const [focused, keydown] = useRovingTabIndex(props.rounds.length);
</script>

<template>
  <nav @keydown="keydown" class="stop">
    <a v-for="round of rounds" :to="`#${round.anchor}`"
      @click="(e) => (e.preventDefault(), goto(round))"
      :tabindex="round.id == rounds[focused.index].id ? 0 : -1"
      :aria-label="`To ${round.name}`" :style="`--hue: ${round.hue}deg;`">
      <span :class="round.complete && 'complete'">{{ round.emoji }}&#xfe0f;</span>
    </a>
  </nav>
</template>

<style scoped>
/* Layout */
nav {
  width: 2.75rem;
  height: calc(100vh - var(--header-height));
  position: sticky;
  top: var(--header-height);
  margin: 0 0 -100vh calc(-1 * var(--nav-margin));

  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 0.4rem;

  /* tooltip needs to appear above round pills */
  z-index: 25;
  overflow: hidden;
}

/* Theming */
nav {
  background-color: white;
  border-right: 1px solid oklch(95% 0.03 275deg);
}

a {
  text-align: center;
  text-decoration: none;
  cursor: pointer;
  user-select: none;
}
</style>
