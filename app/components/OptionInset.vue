<script setup lang="ts">
const { id } = defineProps<{ id: number; }>();
const emit = defineEmits<{ (e: "close"): void; }>();
const { puzzles, updatePuzzleOptimistic } = usePuzzles();

const puzzle = computed(() => puzzles.get(id));

const input = useTemplateRef("input");
const answering = ref<Status | null>(null);
const answer = ref("");

onMounted(() => answer.value = puzzle.value?.answer || "");

const set = (status: Status) => {
  if (StatusNeedsAnswer(status)) {
    answering.value = status;
    nextTick(() => input.value?.focus());
  } else {
    updatePuzzleOptimistic(id, { status, answer: "" });
    emit("close");
  }
};
const submit = () => {
  if (!answering.value || !answer.value) return;
  updatePuzzleOptimistic(id, {
    status: answering.value, answer: answer.value,
  });
  emit("close");
};
const keydown = (e: KeyboardEvent) => {
  if (e.key === "Escape") emit("close");
  else if (e.key === "Enter") submit();
};
</script>

<template>
  <fieldset @keydown="keydown">
    <ETooltip v-if="!answering" v-for="status of Object.values(Status)"
      :text="StatusLabel(status)" placement="top" :offset-distance="4">
      <button :disabled="puzzle?.status === status" @click="() => set(status)">
        {{ StatusEmoji(status) || "❎" }}
      </button>
    </ETooltip>
    <template v-else>
      <input ref="input" type="text" v-model="answer" />
      <button type="submit" @click="submit">
        <UIcon name="i-heroicons-check" />
      </button>
    </template>
  </fieldset>
</template>

<style scoped>
fieldset {
  padding: 0 1.1em;
  border: 1px solid #e1e3e1;
  border-radius: 6px;
  height: 32px;
  background-color: rgb(249 251 253 / 75%);

  display: flex;
  align-self: flex-start;
  gap: 8px;
}

button[disabled] {
  filter: grayscale(100%) opacity(60%);
  pointer-events: none;
}

input {
  font-family: "IBM Plex Mono", "Noto Color Emoji", monospace;
  text-transform: uppercase;
  width: 125px;
}

input,
button[type="submit"] {
  margin: 4px 0;
}
</style>
