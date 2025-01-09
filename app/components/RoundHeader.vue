<script setup lang="ts">
const props = defineProps<{
  round: AnnotatedRound;
  first: boolean;
  filter: boolean;
  timeline: string;
  nextTimeline: string | undefined;
  observer: IntersectionObserver | undefined;
}>();
const emit = defineEmits<{ (e: "edit"): void; }>();
const store = usePuzzles();
const toast = useToast();

const hue = computed(() => props.round.hue);

let registered = false;
const pill = useTemplateRef("pill");
const titles = useTemplateRef("titles");
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

const copy = async (): Promise<void> => {
  const puzzles = store.puzzlesByRound.get(props.round.id);
  if (!puzzles) {
    toast.add({
      title: "No puzzles to copy",
      color: "red",
      icon: "i-heroicons-exclamation-triangle",
    });
    return;
  };

  const data = puzzles.map((p) =>
    [p.name, p.answer ? `<b>${p.answer}</b>` : `${StatusEmoji(p.status)} ${StatusLabel(p.status)}`]);
  const text = data.map(([a, b]) => `${a}, ${b}`).join("\n");
  const html = "<table style='font-family: \"Consolas\";'>\n" + data.map(([a, b]) =>
    `<tr><td>${a}</td><td>${b}</td></tr>`
  ).join("\n") + "</table>";
  await navigator.clipboard.write([
    new ClipboardItem({
      "text/plain": new Blob([text], { type: "text/plain" }),
      "text/html": new Blob([html], { type: "text/html" }),
    }),
  ]);
  toast.add({
    title: "Copied",
    color: "green",
    icon: "i-heroicons-clipboard-document-check",
  });
};
</script>

<template>
  <span class="spacer" :id="round.anchor" v-if="!first"></span>
  <header ref="pill" :class="['pill', props.nextTimeline && 'next', filter && 'filter']">
    <div class="emoji">{{ round.emoji }}&#xfe0f;</div>
    <div class="round">{{ round.name }}</div>
    <div class="flex-spacer"></div>
    <div class="buttons">
      <button @click="copy" v-if="!filter">
        <UIcon name="i-heroicons-clipboard-document-list" size="1rem" />
      </button>
      <button @click="() => emit('edit')">
        <UIcon name="i-heroicons-pencil" size="1rem" />
      </button>
    </div>
    <div class="progress" v-if="!filter">
      {{ round.solved }}/{{ round.total }}
    </div>
    <div class="asterisk" v-if="filter">
      ※
    </div>
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
  scroll-margin-top: calc(-1 * (var(--pill-height-outer) + var(--scroll-fudge) + 1.75rem));
}

.pill {
  grid-column: 1 / 3;
  width: 92.5%;
  margin: 0 0 0.8rem;

  position: sticky;
  top: var(--header-stop);
  z-index: 20;

  height: var(--pill-height);
  line-height: 2.375rem;

  display: flex;
  gap: 0.6rem;
}

.buttons {
  width: 0;
  overflow: hidden;
  flex-shrink: 0;
  opacity: 0%;

  display: flex;
  gap: 0.25rem;
}

.pill:hover .buttons,
.buttons:focus-within {
  width: auto;
  opacity: 100%;
}

.progress,
.asterisk {
  width: 1.375rem;
  margin-left: -0.35rem;
  text-align: right;
}

.titles {
  grid-column: 3 / 6;
  display: grid;
  grid-template-columns: subgrid;
  align-self: flex-end;

  margin-right: 2.5rem;
  padding-bottom: 0.25rem;

  position: sticky;
  top: calc(var(--header-stop) + 0.1rem);
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

  color: oklch(51% 0.075 v-bind(hue));
  background: linear-gradient(white, white) padding-box,
    linear-gradient(68deg,
      oklch(80% 0.10 calc(v-bind(hue) - 10)),
      oklch(65% 0.21 calc(v-bind(hue) + 20))) border-box;

  border: var(--pill-border) solid transparent;
  border-radius: 7px;
  filter: drop-shadow(0 -1px 1px oklch(70% 0.07 v-bind(hue) / 20%));

  cursor: default;
}

.pill.filter {
  color: oklch(51% 0 0deg);
  background: linear-gradient(white, white) padding-box,
    linear-gradient(68deg,
      oklch(80% 0 0deg),
      oklch(65% 0 0deg)) border-box;
}

.pill.filter .emoji,
.pill.filter button {
  filter: grayscale(100%);
}

.round {
  font-weight: 650;

  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.buttons {
  padding: 0 3px;
}

button {
  width: 1.25rem;
  margin: 0.33rem 0;
  line-height: 0.75rem;
  color: oklch(50% 0.21 calc(v-bind(hue) + 20));
}

button:hover {
  color: oklch(30% 0.21 calc(v-bind(hue) + 20));
}

.progress {
  font-variant-numeric: diagonal-fractions;
  color: oklch(50% 0.21 calc(v-bind(hue) + 20));
  opacity: 90%;
  user-select: none;
}

.asterisk {
  font-feature-settings: "case";
  font-size: 1.375rem;
  font-weight: 600;

  line-height: 2.25rem;
  color: oklch(72% 0.19 245deg / 66%);
  user-select: none;
}

.titles {
  font-size: 0.78125rem;
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
    animation-range-start: cover calc(100vh - var(--header-stop) - 2.35rem - 6px);
    animation-range-end: cover calc(100vh - var(--header-stop) - 6px);
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
