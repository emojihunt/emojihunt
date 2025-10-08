<script setup lang="ts">
const { id } = defineProps<{ id: number; }>();
const emit = defineEmits<{ (e: "close"): void; }>();
const { puzzles, voiceRooms, updatePuzzleOptimistic } = usePuzzles();

const puzzle = computed(() => puzzles.get(id));

const set = (voice_room: string) => {
  updatePuzzleOptimistic(id, { voice_room });
  emit("close");
};
</script>

<template>
  <div class="bubble">
    <ETooltip text="No Voice Room" side="top" :side-offset="4">
      <button :disabled="!puzzle?.voice_room" @click="() => set('')">
        ❎
      </button>
    </ETooltip>
    <ETooltip v-for="room of voiceRooms.values()" :text="room.name" side="top"
      :side-offset="4">
      <button :disabled="puzzle?.voice_room === room.id" @click="() => set(room.id)">
        {{ room.emoji }}
      </button>
    </ETooltip>
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
</style>
