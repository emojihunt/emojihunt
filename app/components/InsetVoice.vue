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
  <fieldset>
    <ETooltip text="No Voice Room" placement="top" :offset-distance="4">
      <button :disabled="!puzzle?.voice_room" @click="() => set('')">
        ❎
      </button>
    </ETooltip>
    <ETooltip v-for="room of voiceRooms.values()" :text="room.name" placement="top"
      :offset-distance="4">
      <button :disabled="puzzle?.voice_room === room.id" @click="() => set(room.id)">
        {{ room.emoji }}
      </button>
    </ETooltip>
  </fieldset>
</template>

<style scoped>
fieldset {
  padding: 0 17px;
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
</style>
