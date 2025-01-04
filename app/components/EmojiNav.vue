<script setup lang="ts">
const props = defineProps<{
  rounds: AnnotatedRound[];
  observer: IntersectionObserver | undefined;
}>();
const store = usePuzzles();
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
      <span>{{ round.emoji }}&#xfe0f;</span>
      <label v-if="!round.complete">•</label>
    </a>
  </nav>
</template>

<style scoped>
/* Layout */
nav {
  display: flex;
  gap: 0.4rem;
}

a {
  display: flex;
  flex-direction: column;
}

span {
  display: block;
  width: 1.75rem;
  line-height: 1.75rem;
  height: 2rem;
}

label {
  display: block;
  width: 100%;
  height: 0.75rem;
  margin: -0.2rem 0 0.1rem;
  pointer-events: none;
}

/* Theming */
a {
  border: 1.5px solid transparent;
  border-radius: 3px;
  text-decoration: none;
  cursor: pointer;
}

span {
  text-align: center;
}

label {
  color: oklch(100% 0.08 var(--hue));
  font-size: 0.66rem;
  text-align: center;

  border-radius: 0.3rem;
  z-index: 3;
}

a:hover,
a:focus {
  outline: 2px solid oklch(95% 0.10 var(--hue) / 90%) !important;
  background-color: oklch(95% 0.10 var(--hue) / 50%);
}
</style>
