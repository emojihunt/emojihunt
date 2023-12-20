<script setup lang="ts">
const props = defineProps<{ open: boolean; }>();
const emit = defineEmits<{ (event: "close"): void; }>();
const state = usePuzzles();
const toast = useToast();

const modal = ref();
const data = reactive({ emoji: "", name: "", hue: 274 });
let previous: string;
const submit = () => {
  if (previous) toast.remove(previous);
  state.addRound({ ...data, special: false })
    .then(() => close())
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
  [data.emoji, data.name, data.hue] = ["", "", 274];
  emit("close");
};

watch([props], () => props.open && nextTick(
  () => modal.value?.contents?.querySelector("input")?.focus(),
));
const hue = computed(() => data.hue);
</script>

<template>
  <Modal ref="modal" :open="open" @submit="submit">
    <UForm :state="state" @keydown="(e: KeyboardEvent) => e.key == 'Escape' && close()">
      <UInput v-model="data.emoji" placeholder="🫥" />
      <UInput v-model="data.name" placeholder="Round Name" />
      <URange v-model="data.hue" :min=0 :max="359" class="hue" />
      <UButton label="Add" type="submit" />
    </UForm>
  </Modal>
</template>

<style scoped>
/* Layout */
form {
  display: grid;
  grid-template-columns: 2.5rem 12rem 3.5rem;
  gap: 0.5rem;
}

form div:first-child :deep(input) {
  text-align: center;
}

.hue {
  grid-column: 1 / 4;
}

button {
  grid-row: 1;

  grid-column: 3;
  display: flex;
  justify-content: center;
}

/* Theming */
form button,
.hue :deep(span) {
  background-color: oklch(71% 0.18 v-bind(hue));
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
