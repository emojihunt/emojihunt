<script setup lang="ts">
const props = defineProps<{ id?: number; }>();
const emit = defineEmits<{ (event: "close"): void; }>();
const store = usePuzzles();
const toast = useToast();

const dummy = {
  name: "", answer: "", round: "0", status: Status.NotStarted, note: "",
  location: "", puzzle_url: "", spreadsheet_id: "", discord_channel: "",
  voice_room: "", reminder_date: "", reminder_time: "", meta: false,
  delete: false,
};
const initial = () => {
  if (!props.id) return { ...dummy };
  const puzzle = store.puzzles.get(props.id);
  if (!puzzle) {
    // puzzle was deleted
    emit('close');
    return { ...dummy };
  }
  const date = new Date(puzzle.reminder);
  let reminder_date = "";
  let reminder_time = "";
  if (date.getTime() >= 1700000000000) {
    const parts = new Intl.DateTimeFormat("en-us", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      hour: "2-digit",
      minute: "2-digit",
      hour12: false,
      timeZone: "America/New_York",
    }).formatToParts(date);
    const p: Record<string, string> = {};
    for (const part of parts) p[part.type] = part.value;
    reminder_date = `${p.year}-${p.month}-${p.day}`;
    reminder_time = `${p.hour}:${p.minute}`;
  }
  return {
    ...puzzle,
    round: puzzle.round.id.toString(),
    delete: false,
    reminder_date,
    reminder_time,
  };
};
const data = reactive(initial());
const original = reactive(initial());
watch([props], () => {
  if (props.id) focus();
  for (const [key, value] of Object.entries(initial())) {
    // @ts-ignore
    data[key] = value;
    // @ts-ignore
    original[key] = value;
  }
});

const modified = (key: keyof typeof dummy | "reminder"): Ref<string> =>
  computed(() => {
    let changed = false;
    if (key === "reminder") {
      if (original.reminder_date !== data.reminder_date) changed = true;
      if (original.reminder_time !== data.reminder_time) changed = true;
    } else {
      changed = (original[key] !== data[key]);
    }
    return changed ? "modified" : "";
  });

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
const hue = computed(() => store.rounds.find((r) => r.id === parseInt(data.round))?.hue || 0);
const x = computed(() => console.log("UPDATE", data.round));
</script>

<template>
  <Modal ref="modal" :open="!!id" class="modal-fixup"
    @keydown="(e: KeyboardEvent) => e.key == 'Escape' ? close() : e.stopPropagation()">
    <h1>Puzzle #{{ id }}</h1>
    <form>
      <label for="puzzle-name" :class="modified('name').value">Name</label>
      <UInput v-model="data.name" id="puzzle-name" />
      <label for="puzzle-answer" :class="modified('answer').value">Answer</label>
      <UInput v-model="data.answer" id="puzzle-answer" />
      <label for="puzzle-round" :class="modified('round').value">Round</label>
      <USelect v-model="data.round" :options="rounds" id="puzzle-round" />
      <label for="puzzle-status" :class="modified('status')">Status</label>
      <USelect v-model="data.status" :options="statuses" id="puzzle-status" />
      <label for="puzzle-note" :class="modified('note').value">Note</label>
      <UInput v-model="data.note" id="puzzle-note" />
      <label for="puzzle-location" :class="modified('location').value">Location</label>
      <UInput v-model="data.location" id="puzzle-location" />
      <label for="puzzle-url" :class="modified('puzzle_url').value">Puzzle URL</label>
      <UInput v-model="data.puzzle_url" id="puzzle-url" />
      <label for="puzzle-spreadsheet" :class="modified('spreadsheet_id').value">Spreadsheet
        ID</label>
      <UInput v-model="data.spreadsheet_id" id="puzzle-spreadsheet" />
      <label for="puzzle-channel" :class="modified('discord_channel').value">Text
        Channel ID</label>
      <UInput v-model="data.discord_channel" id="puzzle-channel" />
      <label for="puzzle-voice" :class="modified('voice_room').value">Voice Room
        ID</label>
      <UInput v-model="data.voice_room" id="puzzle-voice" />
      <label for="puzzle-reminder-date" :class="modified('reminder').value">Reminder
        (ET)</label>
      <div class="reminders">
        <UInput v-model="data.reminder_date" id="puzzle-reminder-date" type="date" />
        <UInput v-model="data.reminder_time" id="puzzle-reminder-time" type="time" />
      </div>
      <fieldset>
        <UCheckbox v-model="data.meta" label="Meta" class="checkbox"
          :class="modified('meta').value" />
        <div class="spacer"></div>
        <button class="delete">Delete</button>
        <UButton>Update</UButton>
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
  grid-template-columns: 7rem 16rem;
  gap: 0.5rem;
}

label {
  align-self: flex-end;
  padding-bottom: 0.4rem;
}

.reminders {
  display: flex;
  gap: 0.25rem;
}

fieldset {
  grid-column: 2;
  display: flex;
  gap: 1rem;
  align-items: center;
  justify-content: space-between;
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
