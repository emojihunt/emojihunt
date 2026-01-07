<script setup lang="ts">
const { id } = defineProps<{ id: number; }>();
const toast = useToast();

const { presence, puzzles, users, voiceRooms, updatePuzzleOptimistic } = usePuzzles();
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
    .catch(() => toast.add({
      title: "Error", color: "error", description: "Failed to save puzzle",
      icon: "i-heroicons-exclamation-triangle",
    }))
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
    })
    .catch(() => toast.add({
      title: "Error", color: "error", description: "Failed to save puzzle",
      icon: "i-heroicons-exclamation-triangle",
    }))
    .finally(() => savingRoom.value = false);
};

const items = computed(() => {
  const array = [...[...voiceRooms.values()].map((v => ({ ...v, right: false })))];
  if (!puzzle.location) {
    array.push({ id: "", emoji: "", name: puzzle.voice_room ? "Add In-Person" : "In-Person", right: true });
  }
  return array;
});

const select = (option: string) => {
  if (option) { // voice room
    saveRoom(option);
  } else { // (add) in-person
    if (expanded) expanded.value = 0;
    answering.value = true;
    nextTick(() => location.value?.focus());
  }
};

const MAX_AVATARS = 13;
const present = computed(() => {
  const raw = presence.get(puzzle.id);
  if (!raw) return [];
  const present = [...raw.entries()]
    .filter(([user, _]) => users.has(user))
    .map(([user, active]) => ({ ...users.get(user)!, active }));
  present.sort((a, b) => {
    if (a.active !== b.active) return a.active ? -1 : 1;
    return a.username.localeCompare(b.username);
  });
  return present;
});
</script>

<template>
  <div class="cell">
    <div class="row" :class="(expanded === id) && 'open'">
      <button :class="['room', !(puzzle.location || answering) && 'expand']"
        :data-tabsequence="7" @click="() => expanded = (expanded === id ? 0 : id)">
        <template v-if="room">
          <ETooltip v-if="room" :text="`in ${room.name}`" :offset="6">
            <span class="emoji">{{ room.emoji }}</span>
          </ETooltip>
          <span class="description" v-if="!(puzzle.location || answering)">{{
            room.name }}</span>
        </template>
        <ETooltip v-else-if="puzzle.location" text="Add a Voice Room" :offset="6">
          <span class="emoji">📍</span>
        </ETooltip>
        <ETooltip v-else text="Set a Voice Room" :offset="6">
          <span class="emoji empty">📻</span>
        </ETooltip>
      </button>
      <EditableSpan v-if="puzzle.location || answering" ref="location"
        :value="puzzle.location" :tabsequence="7" @save="saveLocation"
        @cancel="cancelLocation" :readonly="savingText" />
      <button class="clear" v-if="room && !savingRoom && !savingText"
        @click="() => saveRoom('')" tabindex="-1">Clear</button>
      <Spinner v-if="savingText || savingRoom" class="spinner" />
    </div>
    <div class="row presence">
      <ETooltip v-for="[i, user] of present.slice(0, MAX_AVATARS).entries()"
        :text="user.username" side="top" :offset="2" :style="`--sequence: ${i}`">
        <img :src="user.avatarUrl" :class="user.active || 'inactive'" />
      </ETooltip>
      <ETooltip v-if="present.length > MAX_AVATARS" side="top" :offset="2"
        :text="present.slice(MAX_AVATARS).map(x => x.username).join(', ')">
        <span>+{{ present.length - MAX_AVATARS }}</span>
      </ETooltip>
    </div>
    <div class="row">
      <OptionPane v-if="expanded === id" :items="items" @select="select" />
    </div>
  </div>
</template>

<style scoped>
/* Layout */
.cell {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.row {
  display: flex;
  flex-shrink: 0;
}

.row:first-child {
  flex-grow: 1;
  max-width: 100%;
}

.row:last-child {
  flex-basis: 100%;
  display: flex;
  flex-direction: column;
}

button.room {
  display: flex;
}

.emoji {
  display: inline-block;
  width: 1rem;
  text-align: center;
}

button.expand {
  flex-grow: 1;
}

button.clear {
  padding: 3px 0.33rem;
  align-self: flex-start;

  color: oklch(60% 0.15 245deg);
  display: none;
}

.row:first-child:hover button.clear,
.open button.clear,
button.clear:hover,
button.clear:focus {
  display: block;
}

.description {
  padding: 3px 0.33rem;
  white-space: preserve nowrap;
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

button.room {
  height: 28.33px;
  line-height: 28px;
}

button.room:focus-visible {
  /* make Chrome use square outline */
  outline: 2px solid black;
}

.emoji {
  margin: 0 2px;
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

.presence {
  display: flex;
  align-items: center;
  justify-content: flex-end;

  padding: 1px 0;
}

.presence>div {
  flex-shrink: 0;
  margin-right: -7px;
  z-index: calc(32 - var(--sequence));

  width: 24px;
  height: 24px;
  padding: 1.5px;

  border-radius: 50%;
  background-color: white;
  overflow: hidden;
}

.presence>div:last-of-type {
  margin-right: 0;
}

.presence img,
.presence span {
  border-radius: 50%;
}

.presence img.inactive {
  opacity: 50%;
}

.presence span {
  display: block;
  width: 100%;
  height: 100%;

  background: oklch(92.5% 0 0deg);
  line-height: 22px;
  font-size: 10px;
  font-weight: 500;
  text-align: center;
}
</style>
