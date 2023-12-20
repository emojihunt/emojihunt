<script setup lang="ts">
const props = defineProps<{ kind: "round" | "puzzle" | null; open: boolean; }>();
const emit = defineEmits<{ (event: "close"): void; }>();
const state = usePuzzles();
const toast = useToast();

const modal = ref();
const initial = () => ({
  emoji: "", name: "", hue: 274, url: "",
  round: state.rounds.length ? state.rounds[0] : undefined,
});
const data = reactive(initial());

let previous: string;
const submit = () => {
  if (previous) toast.remove(previous);
  const request = (props.kind === "round") ?
    state.addRound({ ...data, special: false }) :
    state.addPuzzle({ name: data.name, round: data.round!.id, puzzle_url: data.url });
  request.then(() => close())
    .catch((e) => (
      previous = toast.add({
        title: "Error",
        color: "red",
        description: e.data.message,
        icon: "i-heroicons-exclamation-triangle",
      }).id),
    );
};
const close = () => {
  if (previous) toast.remove(previous);
  for (const [key, value] of Object.entries(initial())) {
    // @ts-ignore
    data[key] = value;
  }
  emit("close");
};

const focus = () => nextTick(() =>
  modal.value?.contents?.querySelector(
    props.kind === "round" ? "input" : "button"
  )?.focus(),
);
watch([props], () => props.open && focus());

const hue = computed(() => props.kind === "round" ? data.hue : data.round!.hue);
</script>

<template>
  <Modal ref="modal" :open="open" @submit="submit">
    <UForm :state="state" :class="kind"
      @keydown="(e: KeyboardEvent) => e.key == 'Escape' && close()">
      <template v-if="kind === 'round'">
        <UInput v-model="data.emoji" placeholder="🫥" />
        <UInput v-model="data.name" placeholder="Round Name" />
        <URange v-model="data.hue" :min=0 :max="359" class="hue" />
      </template>
      <template v-if="kind === 'puzzle'">
        <USelectMenu v-model="data.round" placeholder="Round" :options="state.rounds"
          option-attribute="displayName" :popper="{ arrow: false }" searchable
          @close="focus">
          <template #trailing>
            <UIcon name="i-heroicons-chevron-up" class="text-gray-400" />
          </template>
        </USelectMenu>
        <UInput v-model="data.name" placeholder="Puzzle Name" />
        <UInput v-model="data.url" placeholder="Puzzle URL" />
      </template>
      <UButton label="Add" type="submit" />
    </UForm>
  </Modal>
</template>

<style scoped>
/* Layout */
form {
  display: grid;
  gap: 0.5rem;
}

form.round {
  grid-template-columns: 2.5rem 12rem 3.5rem;
}

form.puzzle {
  grid-template-columns: 10rem 12rem 12rem 3.5rem;
}

form div:first-child :deep(input) {
  text-align: center;
}

.hue {
  grid-column: 1 / 4;
}

button {
  display: flex;
  justify-content: center;
}

.round button {
  grid-row: 1;
  grid-column: 3;
}

/* Theming */
form button,
.hue :deep(span) {
  background-color: oklch(71% 0.18 v-bind(hue)) !important;
}

form button:hover {
  filter: brightness(90%);
}

form button:focus-visible {
  outline-color: oklch(71% 0.18 v-bind(hue));
}

form :deep(input):focus,
form :deep(button):focus {
  --tw-ring-color: oklch(71% 0.18 v-bind(hue));
}

.hue :deep(input) {
  color: oklch(71% 0.18 v-bind(hue));
}
</style>
