<script setup lang="ts">
import emojifile from "emoji-mart-vue-fast/data/all.json";
import { Picker, EmojiIndex } from "emoji-mart-vue-fast/src";
import emojihues from "~/assets/emoji-hues.json";

const index = new EmojiIndex(emojifile, { recent: [] });

const props = defineProps<{ kind: "round" | "puzzle" | null; open: boolean; }>();
const emit = defineEmits<{ (event: "close"): void; }>();
const store = usePuzzles();
const toast = useToast();

const modal = ref();
const initial = () => ({
  emoji: "", name: "", hue: 274, url: "", create: true,
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
    let request;
    if (props.kind === "round") {
      request = store.addRound({
        name: data.name, emoji: data.emoji, hue: data.hue,
        drive_folder: "+", // *don't* add category yet, to avoid clutter
      });
    } else {
      let params: NewPuzzle = {
        name: data.name, round: data.round!.id, puzzle_url: data.url,
      };
      if (data.create) {
        params = { ...params, spreadsheet_id: "+", discord_channel: "+" };
      }
      request = store.addPuzzle(params);
    }
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
  if (data.emoji === e.native) {
    data.emoji = "";
  } else {
    data.emoji = e.native;
    for (const [hue, ...emojis] of emojihues) {
      if (emojis.includes(e.native)) {
        data.hue = (hue as number);
        break;
      }
    }
  };

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
    <form :class="kind"
      @keydown="(e: KeyboardEvent) => e.key == 'Escape' ? close() : e.stopPropagation()">
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
        <UTooltip text="Create spreadsheet and Discord channel" :open-delay="500"
          :popper="{ placement: 'top', offsetDistance: 0, strategy: 'absolute' }"
          class="checkbox">
          <UCheckbox v-model="data.create" />
        </UTooltip>
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

button {
  display: flex;
  justify-content: center;
}

form.round {
  grid-template-columns: 2.5rem 12rem 3.5rem;
}

form.puzzle {
  grid-template-columns: 10rem 12rem 12rem 1.5rem 3.5rem;
}

.checkbox {
  justify-content: center;
  align-items: center;
}

form.round div:first-child :deep(input) {
  text-align: center;
}

.hue {
  grid-column: 1 / 4;
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

.hue :deep(input),
.checkbox :deep(input) {
  color: oklch(71% 0.18 v-bind(hue));
}

:deep(.emoji-mart-category-label) {
  color: oklch(60% 0.18 v-bind(hue));
}

/* fix for <Modal> capturing pointer events across full width of screen */
footer {
  pointer-events: none;
}

footer :deep(section) {
  pointer-events: auto;
}
</style>
