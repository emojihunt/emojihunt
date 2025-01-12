<script setup lang="ts">
const { id } = defineProps<{ id: number; }>();

const { puzzles, voiceRooms, updatePuzzleOptimistic } = usePuzzles();
const puzzle = puzzles.get(id)!;
const room = computed(() => voiceRooms.get(puzzle.voice_room));

const location = useTemplateRef("location");

const expanded = inject(ExpandedKey);
const savingText = ref(false);
const savingRoom = ref(false);
const answering = ref(false);

const saveLocation = (updated: string) => {
  savingText.value = true;
  updatePuzzleOptimistic(id, { location: updated })
    .then(() => (answering.value = false))
    .finally(() => (savingText.value = false));
};
const cancelLocation = () => (answering.value = false);
const saveRoom = (updated: string) => {
  if (expanded) expanded.value = 0;
  savingRoom.value = true;
  updatePuzzleOptimistic(id, { voice_room: updated })
    .then(() => {
      // add a synthetic delay to reflect sync time
      return new Promise(resolve => setTimeout(resolve, 3500));
    }).finally(() => savingRoom.value = false);
};

const options = computed(() =>
  [...voiceRooms.values(), { id: "", emoji: "+", name: "In-person" }]
);
const select = (option: string) => {
  if (option) { // voice room
    saveRoom(option);
  } else { // "+ In-person"
    if (expanded) expanded.value = 0;
    answering.value = true;
    nextTick(() => location.value?.focus());
  }
};
</script>

<template>
  <div class="cell">
    <div class="row">
      <button :class="['room', !(puzzle.location || answering) && 'expand']"
        :data-tabsequence="7" @click="() => expanded = (expanded === id ? 0 : id)">
        <template v-if="room">
          <ETooltip v-if="room" :text="`in ${room.name}`">
            <span class="emoji">{{ room.emoji }}</span>
          </ETooltip>
          <span class="description" v-if="!(puzzle.location || answering)">{{ room.name
            }}</span>
        </template>
        <ETooltip v-else text="Set a Voice Room">
          <span class="emoji empty">📻</span>
        </ETooltip>
      </button>
      <EditableSpan v-if="puzzle.location || answering" ref="location"
        :value="puzzle.location" :tabsequence="7" @save="saveLocation"
        @cancel="cancelLocation" />
      <button class="clear" v-if="room && !savingRoom && !savingText"
        @click="() => saveRoom('')" tabindex="-1">Clear</button>
      <Spinner v-if="savingText || savingRoom" class="spinner" />
    </div>
    <OptionPane v-if="expanded === id" :options="options" @select="select" />
  </div>
</template>

<style scoped>
/* Layout */
.cell {
  flex-direction: column;
}

.row,
button.room {
  display: flex;
}

.emoji {
  width: 1.5rem;
  text-align: center;
}

button.expand {
  flex-grow: 1;
}

button.clear {
  width: 0;
  padding: 0;
  border-radius: 0;
  color: oklch(60% 0.15 245deg);
}

.cell:hover button.clear,
button.clear:hover,
button.clear:focus {
  width: auto;
  padding: 0 0.33rem;
}

.description {
  padding: 3px 0.33rem;
}

.spinner {
  right: 0.33rem;
  top: 6px;
}

/* Theming */
.cell {
  font-weight: 400;
  font-size: 0.8125rem;
}

.emoji {
  line-height: 1.75rem;
  filter: opacity(90%);
  user-select: none;
}

.emoji.empty {
  filter: grayscale(100%) opacity(50%);
}

.description,
button.clear {
  line-height: 22px;
  font-size: 12.5px;
}
</style>
