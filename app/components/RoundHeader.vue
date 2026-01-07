<script setup lang="ts">
const { id, sequence, filter } = defineProps<{
  id: number;
  sequence: number;
  filter: boolean;
}>();
const emit = defineEmits<{ (e: "edit"): void; }>();
const toast = useToast();

const { rounds, ordering } = usePuzzles();
const round = computed(() => rounds.get(id)!);
const hue = computed(() => round.value.hue);

const timeline = computed(() => timelineFromSequence(sequence));
const nextTimeline = computed(() => timelineFromSequence(sequence + 1));

const pill = useTemplateRef("pill");
const titles = useTemplateRef("titles");
watchEffect(() => {
  pill.value?.classList.add("ready");
  titles.value?.classList.add("ready");
});

let registered = false;
const observer = inject(ObserverKey);
watchEffect(() => {
  if (!registered && observer?.value && titles.value) {
    observer.value.observe(titles.value);
    registered = true;
  }
});

const copy = async (): Promise<void> => {
  const puzzles = ordering.value.find((r) => r.id === id)?.puzzles || [];
  if (!puzzles.length) {
    toast.add({
      title: "No puzzles to copy",
      color: "error",
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
    color: "success",
    icon: "i-heroicons-clipboard-document-check",
  });
};
</script>

<template>
  <span class="spacer" :id="round.anchor" v-if="sequence > 0"></span>
  <header ref="pill" :class="['pill', filter && 'filter']">
    <div class="emoji">{{ round.emoji }}&#xfe0f;</div>
    <div class="round">{{ round.name }}</div>
    <div class="flex-spacer"></div>
    <div class="buttons">
      <button @click="copy" v-if="!filter" tabindex="-1">
        <UIcon name="i-heroicons-clipboard-document-list" size="1rem" />
      </button>
      <button @click="() => emit('edit')" tabindex="-1">
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
  <header ref="titles" :class="['titles', round.total && 'show']">
    <span>Status &bull; Answer</span>
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
  z-index: 60;

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
  z-index: 60;

  pointer-events: none;
  visibility: hidden;
  pointer-events: none;
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
  padding: 0 18px 0 calc(21.25px - 6px);

  color: oklch(51% 0.075 v-bind(hue));
  background: linear-gradient(white, white) padding-box,
    linear-gradient(68deg,
      oklch(72% 0.10 calc(v-bind(hue) - 10)),
      oklch(62% 0.15 calc(v-bind(hue) + 20))) border-box;

  border: var(--pill-border) solid transparent;
  border-left: 6px solid transparent;
  border-radius: 10px;

  cursor: default;
}

.pill.filter {
  color: oklch(51% 0 0deg);
  background: linear-gradient(white, white) padding-box,
    linear-gradient(68deg,
      oklch(72% 0 0deg),
      oklch(62% 0 0deg)) border-box;
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
  .pill.ready,
  .titles.ready,
  .titles.ready span {
    animation-name: fade-out;
    animation-timing-function: ease-in;
    animation-fill-mode: both;
    animation-range-start: cover calc(100vh - var(--header-stop) - 2.35rem - 6px);
    animation-range-end: cover calc(100vh - var(--header-stop) - 6px);
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

.titles.stuck {
  color: white !important;
  /* cover up the previous round's titles */
  text-shadow: oklch(30% 0 0deg) 0 0 5px;
}
</style>
