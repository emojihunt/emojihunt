<script setup lang="ts">
import { Picker, EmojiIndex } from "emoji-mart-vue-fast/src";

// Custom emoji data file. To generate:
//
//  1. git clone https://github.com/serebrov/emoji-mart-vue
//  2. Update emoji-datasource version in package.json
//  3. npm install && node scripts/build-data.js
//  4. Data is in data/all.json
//
// Current version: Unicode 15.1 (emoji-datasource 15.1.2)
//
import emojifile from "~/assets/emojimart.json";

// Generated from http://localhost:3000/emoji (see console output)
import emojihues from "~/assets/emoji-hues.json";

const index = new EmojiIndex(emojifile, { recent: [] });

const props = defineProps<{ kind: "round" | "puzzle"; }>();
const emit = defineEmits<{ (event: "close"): void; }>();
const store = usePuzzles();
const toast = useToast();

const initial = () => ({
  emoji: "", name: "", hue: 274, url: "", create: true,
  round: store.rounds.length ? store.rounds[0] : { id: 0, hue: 0 },
});
const data = reactive(initial());
const saving = ref(false);

let previous: string;
const submit = (e: Event) => {
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
        name: data.name, round: data.round.id, puzzle_url: data.url,
      };
      if (data.create) {
        params = { ...params, spreadsheet_id: "+", discord_channel: "+" };
      }
      request = store.addPuzzle(params);
    }
    request.
      then(() => (
        toast.add({
          title: `Added ${props.kind}`, color: "green",
          icon: "i-heroicons-check-badge",
        }),
        emit("close")
      )).catch((e) => (
        previous = toast.add({
          title: "Error", color: "red", description: e.data.message,
          icon: "i-heroicons-exclamation-triangle",
        }).id),
      ).finally(() => (saving.value = false));
  }, 500);
};

const form = ref<HTMLFormElement>();
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
  form.value?.querySelectorAll("input")[1].focus();
};
const urlBlur = () => {
  if (data.url && !data.url.startsWith("http")) {
    data.url = "https://" + data.url;
  }
};

// USelectMenu doesn't support autofocus
const autofocus = () => (props.kind === "puzzle") &&
  nextTick(() => form.value?.querySelector("button")?.focus());
onMounted(autofocus);
watch([props], autofocus);
const select = () => form.value?.querySelector("input")?.focus();

const hue = computed(() => props.kind === "round" ? data.hue : data.round?.hue);
</script>

<template>
  <form ref="form" :class="kind" @submit="submit">
    <template v-if="kind === 'round'">
      <UInput v-model="data.emoji" placeholder="🫥" readonly="readonly" />
      <UInput v-model="data.name" placeholder="Round Name" />
      <URange v-model="data.hue" :min=0 :max="359" class="hue" />
    </template>
    <template v-else>
      <USelectMenu v-model="data.round" placeholder="Round" :options="store.rounds"
        option-attribute="displayName" :popper="{ arrow: false }" searchable
        clear-search-on-close @close="select">
        <template #trailing>
          <UIcon name="i-heroicons-chevron-up" class="text-gray-400" />
        </template>
      </USelectMenu>
      <UInput v-model="data.name" placeholder="Puzzle Name" />
      <UInput v-model="data.url" placeholder="Puzzle URL" class="url" @blur="urlBlur" />
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
    <Picker v-if="kind === 'round'" :data="index" autoFocus native :showPreview="false"
      :showCategories="false" :emojiSize="16" emojiTooltip @select="emoji" />
  </form>
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
form {
  --form-hue: v-bind(hue);
}
</style>
