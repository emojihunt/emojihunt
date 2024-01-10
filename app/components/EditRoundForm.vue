<script setup lang="ts">
import { RoundKeys } from '~/utils/types';

const props = defineProps<{ id?: number; }>();
const emit = defineEmits<{ (event: "close"): void; }>();
const store = usePuzzles();
const toast = useToast();

const initial = (): Partial<Omit<Round, "id">> =>
  props.id ? { ...store.rounds.find((r) => r.id === props.id) } : {};
const original = reactive(initial());
const edits = reactive({ ...original });
const modified = computed(() => {
  const modified: Partial<Omit<Round, "id">> = {};
  for (const key of RoundKeys) {
    //  Note: we must use "!=" because inputs convert numbers to strings.
    let value = edits[key] || "";
    if (value === "-") value = "";
    // @ts-ignore
    if (value != original[key]) modified[key] = value;
  }
  return modified;
});

const form = ref<HTMLFormElement>();
watch([props], () => {
  if (props.id) nextTick(() => form.value?.querySelector("input")?.focus());
  const updated = initial();
  Object.assign(original, updated);
  Object.assign(edits, updated);
});
watch([storeToRefs(store).rounds], () => {
  const updated = initial();
  for (const key of RoundKeys) {
    if (updated[key] === original[key]) continue;

    if (key === "hue") edits.hue = updated.hue;
    else if (key === "sort") edits.sort = updated.sort;
    else if (key === "special") edits.special = updated.special;
    else edits[key] = updated[key];
  }
  Object.assign(original, updated);
});

let previous: string;
const saving = ref(false);
const submit = (e: Event) => {
  e.preventDefault();
  if (!props.id) return;
  saving.value = true;
  if (previous) toast.remove(previous);
  store.updateRound(props.id, modified.value)
    .then(() => (
      toast.add({
        title: "Updated round", color: "green",
        icon: "i-heroicons-check-badge",
      }),
      emit("close")
    )).catch((e) => (
      previous = toast.add({
        title: "Error", color: "red", description: e.data.message,
        icon: "i-heroicons-exclamation-triangle",
      }).id),
    ).finally(() => (saving.value = false));
};

const del = (e: MouseEvent) => {
  e.preventDefault();
  if (!props.id) return;
  if (!confirm("Delete this round?")) return;
  if (previous) toast.remove(previous);
  store.deleteRound(props.id)
    .then(() => (
      toast.add({
        title: "Deleted round", color: "green", icon: "i-heroicons-trash",
      }),
      emit("close")
    )).catch((e) => (
      previous = toast.add({
        title: "Error", color: "red", description: e.data.message,
        icon: "i-heroicons-exclamation-triangle",
      }).id),
    ).finally(() => (saving.value = false));
};

const rounds = computed(() => store.rounds.map((r) =>
  ({ label: `${r.emoji}\ufe0f ${r.name}`, value: r.id })));
const statuses = computed(() => Statuses.map((s) =>
  ({ label: `${StatusEmoji(s)} ${StatusLabel(s)}`, value: s })));
const hue = computed(() => edits.hue || 0);
</script>

<template>
  <h1>Round #{{ id }}</h1>
  <form ref="form" @submit="submit">
    <label for="round-name" :class="'name' in modified && 'modified'">Name</label>
    <UInput v-model="edits.name" id="round-name" autofocus />
    <label for="round-emoji" :class="'emoji' in modified && 'modified'">Emoji</label>
    <UInput v-model="edits.emoji" id="round-emoji" />
    <label for="round-hue" :class="'hue' in modified && 'modified'">Hue</label>
    <URange v-model="edits.hue" :min=0 :max="359" id="round-hue" class="hue" />
    <label for="round-sort" :class="'sort' in modified && 'modified'">Sort</label>
    <UInput v-model="edits.sort" id="round-sort" />
    <label for="round-drive" :class="'drive_folder' in modified && 'modified'">Drive
      Folder</label>
    <UInput v-model="edits.drive_folder" id="round-drive" />
    <label for="round-discord" :class="'discord_category' in modified && 'modified'">Discord
      Group</label>
    <UInput v-model="edits.discord_category" id="round-discord" />
    <fieldset>
      <UCheckbox v-model="edits.special" label="Special" class="checkbox"
        :class="'special' in modified && 'modified'" />
      <div class="spacer"></div>
      <button class="delete" type="button" @click="del">Delete</button>
      <UButton type="submit" :disabled="saving">
        <Spinner v-if="saving" />
        <span v-else>Update</span>
      </UButton>
    </fieldset>
  </form>
</template>

<style scoped>
/* Layout */
h1 {
  margin: 0.5rem;
}

form {
  margin: 0 0.5rem;
  display: grid;
  grid-template-columns: 7rem 17rem;
  gap: 0.5rem;
}

label {
  align-self: flex-end;
  padding-bottom: 0.4rem;
}

.reminders {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.25rem;
}

fieldset {
  grid-column: 2;
  display: flex;
  gap: 1rem;
  align-items: center;
  justify-content: space-between;
}

button[type="submit"] {
  width: 4.75rem;
  height: 2rem;
  display: flex;
  justify-content: center;
}

.spacer {
  flex-grow: 1;
}

/* Theming */
h1 {
  font-size: 1rem;
  font-weight: 600;
}

label {
  font-size: 0.85rem;
  user-select: none;
}

label.modified {
  font-weight: 600;
  color: oklch(60% 0.18 v-bind(hue));
}

.checkbox.modified :deep(label) {
  color: oklch(60% 0.18 v-bind(hue)) !important;
}

form button.delete {
  font-weight: 500;
  font-size: 0.9rem;
  padding: 0.25rem;

  color: oklch(60% 0.15 30deg);
  background: none !important;
  border-radius: 2px;
}

form button.delete:hover {
  color: oklch(45% 0.15 30deg);
  filter: none;
}

form button,
.hue :deep(span) {
  background-color: oklch(71% 0.18 v-bind(hue)) !important;
}

form button:hover {
  filter: brightness(90%);
}

form button:focus,
form select:focus {
  outline-color: oklch(71% 0.18 v-bind(hue));
}

form :deep(input):focus,
form :deep(button):focus,
form :deep(select):focus {
  --tw-ring-color: oklch(71% 0.18 v-bind(hue));
}

.hue :deep(input),
.checkbox :deep(input) {
  color: oklch(71% 0.18 v-bind(hue));
}
</style>
