<script setup lang="ts">
import emojifile from "emoji-mart-vue-fast/data/all.json";
import { Picker, EmojiIndex } from "emoji-mart-vue-fast/src";

const index = new EmojiIndex(emojifile, { recent: [] });

const props = defineProps<{ kind: "round" | "puzzle" | null; open: boolean; }>();
const emit = defineEmits<{ (event: "close"): void; }>();
const store = usePuzzles();
const toast = useToast();

const modal = ref();
const initial = () => ({
  emoji: "", name: "", hue: 274, url: "",
  round: store.rounds.length ? store.rounds[0] : undefined,
});
const data = reactive(initial());
const saving = ref(false);

let previous: string;
const submit = (e: SubmitEvent) => {
  e.preventDefault();
  saving.value = true;
  if (previous) toast.remove(previous);
  setTimeout(() => {
    const request = (props.kind === "round") ?
      store.addRound({
        name: data.name, emoji: data.emoji, hue: data.hue,
        drive_folder: "+", // *don't* add category yet, to avoid clutter
      }) :
      store.addPuzzle({
        name: data.name, round: data.round!.id, puzzle_url: data.url,
        spreadsheet_id: "+", discord_channel: "+",
      });
    request.then(() => close())
      .catch((e) => (
        previous = toast.add({
          title: "Error",
          color: "red",
          description: e.data.message,
          icon: "i-heroicons-exclamation-triangle",
        }).id),
      ).finally(() => (saving.value = false));
  }, 500);
};
const close = () => {
  if (previous) toast.remove(previous);
  for (const [key, value] of Object.entries(initial())) {
    // @ts-ignore
    data[key] = value;
  }
  emit("close");
};

const emoji = (e: any) => {
  if (data.emoji === e.native) data.emoji = "";
  else data.emoji = e.native;
  modal.value?.contents.querySelector(".name input")?.focus();
};
const focus = () => nextTick(() =>
  modal.value?.contents?.querySelector(
    props.kind === "round" ? ".emoji-mart input" : "button"
  )?.focus(),
);
watch([props], () => props.open && focus());

const hue = computed(() => props.kind === "round" ? data.hue : data.round?.hue);
</script>

<template>
  <Modal ref="modal" :open="open" @submit="submit">
    <form :class="kind" @keydown="(e: KeyboardEvent) => e.key == 'Escape' && close()">
      <template v-if="kind === 'round'">
        <UInput v-model="data.emoji" placeholder="🫥" readonly="readonly" />
        <UInput v-model="data.name" placeholder="Round Name" class="name" />
        <URange v-model="data.hue" :min=0 :max="359" class="hue" />
      </template>
      <template v-else>
        <USelectMenu v-model="data.round" placeholder="Round" :options="store.rounds"
          option-attribute="displayName" :popper="{ arrow: false }" searchable
          @close="focus">
          <template #trailing>
            <UIcon name="i-heroicons-chevron-up" class="text-gray-400" />
          </template>
        </USelectMenu>
        <UInput v-model="data.name" placeholder="Puzzle Name" />
        <UInput v-model="data.url" placeholder="Puzzle URL" />
      </template>
      <UButton type="submit" :disabled="saving">
        <Spinner v-if="saving" />
        <span v-else>Add</span>
      </UButton>
      <Picker v-if="kind === 'round'" :data="index" autofocus native :showPreview="false"
        :showCategories="false" :emojiSize="16" emojiTooltip @select="emoji" />
    </form>
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

.emoji-mart {
  grid-column: 1 / 4;
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

:deep(.emoji-mart-category-label) {
  color: oklch(60% 0.18 v-bind(hue));
}
</style>
