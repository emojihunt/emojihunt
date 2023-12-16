<script setup lang="ts">
const props = defineProps<{
  round: RoundStats,
  timeline: string;
  nextTimeline: string | undefined;
  observer: IntersectionObserver | undefined;
}>();
const hue = props.round.hue;

const pill = ref<HTMLElement>();
const titles = ref<HTMLElement>();
onMounted(() => {
  if (props.nextTimeline) {
    pill.value?.classList.add("ready");
    titles.value?.classList.add("ready");
  }
  props.observer?.observe(pill.value!);
  props.observer?.observe(titles.value!);
});
</script>

<template>
  <span class="spacer" :id="round.anchor"></span>
  <header class="pill" ref="pill" :id="round.anchor">
    <div class="emoji">{{ round.emoji }}&#xfe0f;</div>
    <div class="round">{{ round.name }}</div>
    <div class="progress">{{ round.solved }}/{{ round.total }} </div>
  </header>
  <header class="titles" ref="titles">
    <span>Status &bull; Answer</span>
    <span>Location</span>
    <span>Note</span>
  </header>
</template>

<style scoped>
/* Layout */
.spacer {
  height: 2.8rem;
  scroll-margin-block-start: calc(6rem - 1.4rem - 2.8rem);
}

.pill {
  grid-column: 1 / 3;
  width: 83%;
  margin: 0 0 0.8rem;
  display: flex;

  height: 2.25rem;
  line-height: 2.35rem;

  position: sticky;
  top: calc(6rem - 1.4rem);

  z-index: 20;
}

.titles {
  grid-column: 3 / 6;
  display: grid;
  grid-template-columns: subgrid;
  align-self: flex-end;

  padding-bottom: 0.25rem;

  position: sticky;
  top: calc(6rem - 1.4rem + 0.1rem);
  z-index: 20;
}

.titles span {
  padding: 0 0.30rem;
}

/* Theming */
.pill {
  font-size: 1.07rem;
  padding: 0 1.2rem;
  gap: 0.6rem;

  color: oklch(51% 0.075 v-bind(hue));
  border-radius: 0.6rem;
  border: 2.5px solid transparent;
  background: linear-gradient(68deg, oklch(100% 0 0deg / 90%),
      oklch(100% 0 0deg / 84%)) padding-box,
    linear-gradient(68deg,
      oklch(90% 0.10 calc(v-bind(hue) - 15)),
      oklch(65% 0.30 calc(v-bind(hue) + 75))) border-box;
  filter: drop-shadow(0 1px 2px oklch(70% 0.07 v-bind(hue) / 25%));

  cursor: default;
}

.round {
  font-weight: 715;
}

.progress {
  flex-grow: 1;
  text-align: right;
  font-variant-numeric: diagonal-fractions;
  color: oklch(50% 0.30 calc(v-bind(hue) + 75));
  user-select: none;
}

.titles {
  font-size: 0.8rem;
  font-weight: 430;
  color: oklch(55% 0.03 275deg);

  user-select: none;
}

/* Animation */
@supports(view-timeline: --test) {
  .pill {
    view-timeline: v-bind(timeline);
  }

  /* FYI, if we use the `animation` shorthand propety, Nuxt may incorrectly
     re-order it with other `animation-*` properties. */
  .pill.ready,
  .titles.ready,
  .titles.ready span {
    animation-name: fade-out;
    animation-timing-function: ease-in;
    animation-fill-mode: both;
    animation-range-start: cover calc(100vh - (6rem - 1.4rem) - 2.35rem - 6px);
    animation-range-end: cover calc(100vh - (6rem - 1.4rem) - 6px);
    animation-timeline: v-bind(nextTimeline);
  }

  .titles.ready span {
    animation-name: color-in;
    animation-timing-function: linear;
    animation-timeline: view();
  }
}

.titles {
  /* fallback: avoid flicker when scrolling at medium speed */
  transition: visibility 0.025s;
}

/* see main.css for keyframes and `stuck` styles */
</style>