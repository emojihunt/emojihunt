<script setup lang="ts">
import { DefaultReminder, PuzzleKeys } from '~/utils/types';

const props = defineProps<{ id?: number; }>();
const emit = defineEmits<{ (event: "close"): void; }>();
const store = usePuzzles();
const toast = useToast();

type EditState = Omit<Omit<Omit<Puzzle, "id">, "round">, "reminder"> & {
  round: string; rdate: string; rtime: string;
};
const editState = (puzzle: Partial<Omit<Puzzle, "id">>): Partial<EditState> => {
  let [rdate, rtime] = ["", ""];
  const date = puzzle.reminder && new Date(puzzle.reminder);
  if (date && date.getTime() >= 1700000000000) {
    const parts = new Intl.DateTimeFormat("en-us", {
      year: "numeric", month: "2-digit", day: "2-digit",
      hour: "2-digit", minute: "2-digit", hour12: false,
      timeZone: "America/New_York",
    }).formatToParts(date);
    const p: Record<string, string> = {};
    for (const part of parts) p[part.type] = part.value;
    rdate = `${p.year}-${p.month}-${p.day}`;
    rtime = `${p.hour}:${p.minute}`;
  }
  return { ...puzzle, round: (puzzle?.round || "0").toString(), rdate, rtime };
};

const initial = (): Partial<Omit<Puzzle, "id">> =>
  props.id ? (store.puzzles.get(props.id) || {}) : {};
const original = reactive(initial());
const edits = reactive(editState(original));
const modified = computed(() => {
  const modified: Partial<Omit<Puzzle, "id">> = {};
  let reminder = DefaultReminder;
  if (edits.rdate && edits.rtime) {
    reminder = new Date(
      `${edits.rdate}T${edits.rtime}:00-0500`,
    ).toISOString().replaceAll(".000Z", "Z");
  }
  if (reminder !== original.reminder) modified.reminder = reminder;

  for (const key of PuzzleKeys) {
    if (key === "status") {
      if (edits.status !== original.status) modified.status = edits.status;
    } else if (key === "round") {
      const edited = parseInt(edits.round || "0");
      if (edited !== original.round) modified.round = edited;
    } else if (key === "meta") {
      if (edits.meta !== original.meta) modified.meta = edits.meta;
    } else if (key === "reminder") {
    } else {
      let value = edits[key] || "";
      if (value === "-") value = "";
      if (key === "answer") value = value.toUpperCase();
      if (value !== original[key]) modified[key] = value;
    }
  }
  return modified;
});

watch([props], () => {
  const updated = initial();
  Object.assign(original, updated);
  Object.assign(edits, editState(updated));
  if (props.id) focus();
});
watch([storeToRefs(store).puzzlesByRound], () => {
  const updated = initial();
  const transformed = editState(updated);
  for (const key of PuzzleKeys) {
    if (updated[key] === original[key]) continue;

    if (key === "status") edits.status = transformed.status;
    else if (key === "meta") edits.meta = transformed.meta;
    else if (key === "reminder") {
      edits.rdate = transformed.rdate;
      edits.rtime = transformed.rtime;
    }
    else edits[key] = transformed[key];
  }
  Object.assign(original, updated);
});

const saving = ref(false);
const submit = (e: SubmitEvent) => {
  e.preventDefault();
  if (!props.id) return;
  saving.value = true;
  if (previous) toast.remove(previous);
  store.updatePuzzle(props.id, modified.value)
    .then(() => (
      toast.add({
        title: "Updated", color: "green", icon: "i-heroicons-check-badge",
      }),
      emit("close") // don't dismiss toast
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
  if (!confirm("Delete this puzzle?")) return;
  store.deletePuzzle(props.id)
    .then(() => (
      toast.add({
        title: "Deleted", color: "green", icon: "i-heroicons-trash",
      }),
      emit("close") // don't dismiss toast
    )).catch((e) => (
      previous = toast.add({
        title: "Error", color: "red", description: e.data.message,
        icon: "i-heroicons-exclamation-triangle",
      }).id),
    ).finally(() => (saving.value = false));
};

let previous: string;
const modal = ref();
const focus = () => nextTick(() =>
  modal.value?.contents?.querySelector("#puzzle-name")?.focus(),
);
const close = () => {
  if (previous) toast.remove(previous);
  emit("close");
};

const rounds = computed(() => store.rounds.map((r) =>
  ({ label: `${r.emoji}\ufe0f ${r.name}`, value: r.id })));
const statuses = computed(() => Statuses.map((s) =>
  ({ label: `${StatusEmoji(s)} ${StatusLabel(s)}`, value: s })));
const hue = computed(() => store.rounds.find((r) => r.id === parseInt(edits.round || "0"))?.hue || 0);
</script>

<template>
  <Modal ref="modal" :open="!!id" class="modal-fixup" @submit="submit"
    @keydown="(e: KeyboardEvent) => e.key == 'Escape' ? close() : e.stopPropagation()">
    <h1>Puzzle #{{ id }}</h1>
    <form>
      <label for="puzzle-name" :class="'name' in modified && 'modified'">Name</label>
      <UInput v-model="edits.name" id="puzzle-name" />
      <label for="puzzle-answer" :class="'answer' in modified && 'modified'">Answer</label>
      <UInput v-model="edits.answer" id="puzzle-answer" />
      <label for="puzzle-round" :class="'round' in modified && 'modified'">Round</label>
      <USelect v-model="edits.round" :options="rounds" id="puzzle-round" />
      <label for="puzzle-status" :class="'status' in modified && 'modified'">Status</label>
      <USelect v-model="edits.status" :options="statuses" id="puzzle-status" />
      <label for="puzzle-note" :class="'note' in modified && 'modified'">Note</label>
      <UInput v-model="edits.note" id="puzzle-note" />
      <label for="puzzle-location"
        :class="'location' in modified && 'modified'">Location</label>
      <UInput v-model="edits.location" id="puzzle-location" />
      <label for="puzzle-url" :class="'puzzle_url' in modified && 'modified'">Puzzle
        URL</label>
      <UInput v-model="edits.puzzle_url" id="puzzle-url" />
      <label for="puzzle-spreadsheet"
        :class="'spreadsheet_id' in modified && 'modified'">Spreadsheet
        ID</label>
      <UInput v-model="edits.spreadsheet_id" id="puzzle-spreadsheet" />
      <label for="puzzle-channel" :class="'discord_channel' in modified && 'modified'">Text
        Channel ID</label>
      <UInput v-model="edits.discord_channel" id="puzzle-channel" />
      <label for="puzzle-voice" :class="'voice_room' in modified && 'modified'">Voice Room
        ID</label>
      <UInput v-model="edits.voice_room" id="puzzle-voice" />
      <label for="puzzle-reminder-date"
        :class="'reminder' in modified && 'modified'">Reminder
        (ET)</label>
      <div class="reminders">
        <UInput v-model="edits.rdate" id="puzzle-reminder-date" type="date" />
        <UInput v-model="edits.rtime" id="puzzle-reminder-time" type="time" />
      </div>
      <fieldset>
        <UCheckbox v-model="edits.meta" label="Meta" class="checkbox"
          :class="'meta' in modified && 'modified'" />
        <div class="spacer"></div>
        <button class="delete" type="button" @click="del">Delete</button>
        <UButton type="submit" :disabled="saving">
          <Spinner v-if="saving" />
          <span v-else>Update</span>
        </UButton>
      </fieldset>
    </form>
  </Modal>
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
}

form button.delete:hover {
  color: oklch(45% 0.15 30deg);
  filter: none;
}

form button {
  background-color: oklch(71% 0.18 v-bind(hue)) !important;
}

form button:hover {
  filter: brightness(90%);
}

form button:focus-visible,
form select:focus-visible {
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

:deep(input#puzzle-answer) {
  font-family: "IBM Plex Mono", "Noto Color Emoji", monospace;
  text-transform: uppercase;
}
</style>
