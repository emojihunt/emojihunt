<script setup lang="ts">
const { id } = defineProps<{ id: number; }>();
const emit = defineEmits<{ (e: "close"): void; }>();
const { puzzles, updatePuzzleOptimistic } = usePuzzles();

const puzzle = computed(() => puzzles.get(id));

const input = useTemplateRef("input");
const answering = ref<Status | null>(null);
const answerModel = ref("");

onMounted(() => answerModel.value = puzzle.value?.answer || "");

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
  const answer = answerModel.value?.toUpperCase();
  if (!answering.value || !answer) return;
  updatePuzzleOptimistic(id, { status: answering.value, answer, voice_room: "" });
  emit("close");
};
const keydown = (e: KeyboardEvent) => {
  if (e.key === "Escape") emit("close");
  else if (e.key === "Enter") submit();
};
</script>

<template>
  <div class="bubble" @keydown="keydown">
    <ETooltip v-if="!answering" v-for="status of Object.values(Status)"
      :text="StatusLabel(status)" side="top">
      <button :disabled="puzzle?.status === status" @click="() => set(status)">
        {{ StatusEmoji(status) || "❎" }}
      </button>
    </ETooltip>
    <template v-else>
      <input ref="input" type="text" v-model="answerModel" />
      <button type="submit" @click="submit">
        <UIcon name="i-heroicons-check" />
      </button>
    </template>
  </div>
</template>

<style scoped>
.bubble {
  padding: 0 17px;
  border: 1px solid #e1e3e1;
  border-radius: 6px;
  height: 32px;
  background-color: rgb(249 251 253 / 75%);

  align-self: flex-start;
  flex-shrink: 0;

  display: flex;
  align-items: center;
  gap: 8px;
}

button[disabled] {
  filter: grayscale(100%) opacity(60%);
  pointer-events: none;
}

input {
  font-family: var(--monospace-fonts);
  text-transform: uppercase;
  width: 125px;
}

input,
button[type="submit"] {
  margin: 4px 0;
}
</style>
