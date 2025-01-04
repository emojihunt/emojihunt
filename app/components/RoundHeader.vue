<script setup lang="ts">
const props = defineProps<{
  round: AnnotatedRound,
  timeline: string;
  nextTimeline: string | undefined;
  observer: IntersectionObserver | undefined;
}>();
const emit = defineEmits<{ (e: "copy"): void; (e: "edit"): void; }>();
const hue = computed(() => props.round.hue);

let registered = false;
const pill = ref<HTMLElement>();
const titles = ref<HTMLElement>();
const ready = () => {
  pill.value?.classList.add("ready");
  titles.value?.classList.add("ready");
  if (!registered && props.observer) {
    props.observer.observe(titles.value!);
    registered = true;
  }
};
watch([props], () => nextTick(ready));
onMounted(() => nextTick(ready));
</script>

<template>
  <span class="spacer" :id="round.anchor"></span>
  <header ref="pill" :id="round.anchor"
    :class="['pill', props.nextTimeline ? 'next' : '']">
    <div class="emoji">{{ round.emoji }}&#xfe0f;</div>
    <div class="round">{{ round.name }}</div>
    <div class="spaces"></div>
    <button @click="() => emit('edit')">Edit</button>
    <button @click="() => emit('copy')">Copy</button>
    <div class="progress">{{ round.solved }}/{{ round.total }}</div>
  </header>
  <header ref="titles"
    :class="['titles', round.total && 'show', props.nextTimeline ? 'next' : '']">
    <span>Status &bull;
      Answer</span>
    <span>Location</span>
    <span>Note</span>
  </header>
</template>

<style scoped>
/* Layout */
.spacer {
  height: 1.75rem;
  scroll-margin-block-start: 2.845rem;
}

.spacer:first-of-type {
  display: none;
}

.pill {
  grid-column: 1 / 3;
  width: 92.5%;
  margin: 0 0 0.8rem;
  display: flex;

  height: 2.5rem;
  line-height: 2.35rem;

  position: sticky;
  top: calc(6rem - 1.4rem);

  z-index: 20;
}

button {
  width: 0;
  overflow: hidden;
  flex-shrink: 0;
}

.pill:hover button,
button:focus {
  width: auto;
  padding: 0.25rem;
  margin: 0.25rem 0;
  border-radius: 2px;
}

.spaces {
  flex-grow: 1;
}

.titles {
  grid-column: 3 / 6;
  display: grid;
  grid-template-columns: subgrid;
  align-self: flex-end;

  margin-right: 2.5rem;
  padding-bottom: 0.25rem;

  position: sticky;
  top: calc(6rem - 1.4rem + 0.1rem);
  z-index: 20;

  visibility: hidden;
}

.titles.show {
  visibility: visible;
}

.titles span {
  padding: 0 0.30rem;
}

/* Theming */
.pill {
  font-size: 1rem;
  padding: 0 1.2rem;
  gap: 0.6rem;

  color: oklch(51% 0.075 v-bind(hue));
  background: linear-gradient(white, white) padding-box,
    linear-gradient(68deg,
      oklch(80% 0.10 calc(v-bind(hue) - 10)),
      oklch(65% 0.21 calc(v-bind(hue) + 20))) border-box;

  border: 2.5px solid transparent;
  border-radius: 7px;
  filter: drop-shadow(0 -1px 1px oklch(70% 0.07 v-bind(hue) / 20%));

  cursor: default;
}

.round {
  font-weight: 650;

  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

button {
  font-weight: 550;
  font-size: 0.7rem;
  color: oklch(50% 0.21 calc(v-bind(hue) + 20));
}

button:hover {
  color: oklch(30% 0.21 calc(v-bind(hue) + 20));
}

.progress {
  font-variant-numeric: diagonal-fractions;
  color: oklch(50% 0.21 calc(v-bind(hue) + 20));
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
  .pill.ready.next,
  .titles.ready.next,
  .titles.ready.next span {
    animation-name: fade-out;
    animation-timing-function: ease-in;
    animation-fill-mode: both;
    animation-range-start: cover calc(100vh - (6rem - 1.4rem) - 2.35rem - 6px);
    animation-range-end: cover calc(100vh - (6rem - 1.4rem) - 6px);
    animation-timeline: v-bind(nextTimeline);
  }

  .titles.ready.next span {
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
