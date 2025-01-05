<script setup lang="ts">
const props = defineProps<{
  rounds: AnnotatedRound[];
  observer: IntersectionObserver | undefined;
}>();
const state = usePuzzles();
const url = useRequestURL();

// Navigate to anchors without changing the fragment
const goto = (round: AnnotatedRound) => {
  const id = (new URL(`#${round.anchor}`, url)).hash; // escaping
  document.querySelector<HTMLElement>(`${id} ~ .row [tabIndex='0']`)?.focus();

  // Workaround: sometimes scrolling to the first anchor doesn't work.
  if (round.id == props.rounds[0].id) window.scrollTo({ top: 0 });
  else document.querySelector(id)?.scrollIntoView();

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

const nav = useTemplateRef<HTMLElement>("nav");
const [focused, _] = useRovingTabIndex(props.rounds.length);
const keydown = (e: KeyboardEvent): void => {
  if (e.key === "ArrowUp" || e.key === "ArrowLeft") {
    if (focused.index > 0) focused.index -= 1;
  } else if (e.key === "ArrowDown" || e.key == "ArrowRight") {
    if (focused.index < props.rounds.length - 1) focused.index += 1;
  } else {
    return;
  }
  // @ts-ignore
  nextTick(() => nav.value.querySelector("[tabindex='0']")?.focus());
  e.preventDefault();
  e.stopPropagation();
};
</script>

<template>
  <nav ref="nav" @keydown="keydown">
    <UTooltip text="Puzzles Open" :open-delay="250" class="stats"
      :popper="{ placement: 'right', offsetDistance: -5 }">
      {{ String(state.puzzleCount - state.solvedPuzzleCount).padStart(3, '0') }}
    </UTooltip>
    <p class="dot"></p>
    <UTooltip v-for="round of rounds" :text="round.name" :open-delay="250"
      :popper="{ placement: 'right', offsetDistance: -5 }">
      <a :href="`#${round.anchor}`" @click="(e) => (e.preventDefault(), goto(round))"
        :tabindex="round.id == rounds[focused.index].id ? 0 : -1"
        :aria-label="`To ${round.name}`" :style="`--hue: ${round.hue}deg;`">
        <span :class="round.complete && 'complete'">{{ round.emoji }}&#xfe0f;</span>
      </a>
    </UTooltip>
    <p class="dot"></p>
    <UTooltip text="Puzzles Solved" :open-delay="250" class="stats"
      :popper="{ placement: 'right', offsetDistance: -5 }">
      {{ String(state.solvedPuzzleCount).padStart(3, '0') }}
    </UTooltip>
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
  gap: 0.2rem;

  /* tooltip needs to appear above round pills */
  z-index: 25;
  overflow: hidden;
}

nav>div {
  display: flex;
  flex-direction: column;
  align-items: center;
}

a {
  padding: 3px;
}

p,
.stats {
  display: block;
  text-align: center;
  height: 1em;
}

p.dot {
  margin: -1px 0;
}

/* Theming */
nav {
  background-color: white;
  border-right: 1px solid oklch(95% 0.03 275deg);
}

p,
.stats {
  font-weight: 600;
  font-size: 0.7rem;
  font-variant-numeric: tabular-nums;
  font-feature-settings: 'ss01', 'zero';
  color: oklch(33% 0 0deg);
  cursor: default;
}

p.dot:before {
  content: "\00b7";
}

a {
  text-align: center;
  text-decoration: none;
  outline-color: oklch(33% 0 0deg) !important;

  cursor: pointer;
  user-select: none;
}

a span {
  opacity: 90%;
  display: block;
}

a span.complete {
  opacity: 50%;
  filter: grayscale(100%);

  /* Strikethrough. https://stackoverflow.com/a/40499367 */
  background: linear-gradient(to left top,
      transparent 47.75%, currentColor 49.5%,
      currentColor 50.5%, transparent 52.25%);
}

a:hover span {
  opacity: 100%;
  filter: drop-shadow(0 1px 1px oklch(85% 0 0deg));
  transform: scale(110%);
}

a:hover span.complete {
  opacity: 80%;
}
</style>
