<script setup lang="ts">
const { id } = defineProps<{ id: number; }>();
const emit = defineEmits<{ (event: "close"): void; }>();

const { rounds, updateRound, deleteRound } = usePuzzles();
const toast = useToast();

const original = reactive({ ...rounds.get(id) });
const edits = reactive({ ...original });
const modified = computed(() => {
  const modified: Partial<Omit<Round, "id">> = {};
  for (const key of RoundKeys) {
    let value = edits[key] || "";
    if (value === "-") value = "";
    // @ts-ignore
    // Note: we must use "!=" because inputs convert numbers to strings.
    if (value != original[key]) modified[key] = value;
  }
  return modified;
});

const form = useTemplateRef("form");
watch(() => id, () => {
  Object.assign(original, rounds.get(id));
  Object.assign(edits, rounds.get(id));
  nextTick(() => form.value?.querySelector("input")?.focus());
});
watch(rounds.get(id)!, (updated) => {
  updated = { ...updated };
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
  if (!id) return;
  saving.value = true;
  if (previous) toast.remove(previous);
  updateRound(id, modified.value)
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
  if (!id) return;
  if (!confirm("Delete this round?")) return;
  if (previous) toast.remove(previous);
  deleteRound(id)
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
    <label for="round-discord"
      :class="'discord_category' in modified && 'modified'">Discord
      Group</label>
    <UInput v-model="edits.discord_category" id="round-discord" />
    <fieldset>
      <UCheckbox v-model="edits.special" label="Special" class="checkbox"
        :class="'special' in modified && 'modified'" />
      <div class="flex-spacer"></div>
      <UButton color="red" variant="ghost" @click="del">
        Delete
      </UButton>
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

fieldset {
  grid-column: 2;
  display: flex;
  gap: 1rem;
}

fieldset>:first-child {
  align-self: center;
}

button[type="submit"] {
  width: 4.75rem;
  display: flex;
  justify-content: center;
}

/* Theming */
h1 {
  font-size: 1rem;
  font-weight: 600;
}

form {
  --form-hue: v-bind(hue);
}

label {
  font-size: 0.875rem;
  user-select: none;
}
</style>
