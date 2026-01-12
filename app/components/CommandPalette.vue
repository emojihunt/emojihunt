<script setup lang="ts">
const { ordering } = defineProps<{ ordering: SortedRound[]; }>();
const emit = defineEmits<{ (event: "close", id?: number): void; }>();

const fuse = {
  fuseOptions: {
    ignoreDiacritics: true,
  }
};
const groups = computed(() => ordering.map(round => ({
  id: `round-${round.id}`,
  slot: "puzzles",
  label: `${round.emoji} ${round.name}`,
  items: round.puzzles.map(puzzle => ({
    label: puzzle.name, puzzle, round,
  }))
})));

// If the modal is closed by selecting a puzzle, *don't* restore the previously
// focused element because we're going to navigate away in a moment.
const selected = ref(false);
const onCloseAutoFocus = (e: Event) => selected.value && e.preventDefault();
</script>
<template>
  <UModal title="Command Palette" description="Search for a puzzle"
    :content="{ onCloseAutoFocus }">
    <template #content>
      <UCommandPalette placeholder="Search for a puzzle..." close :groups :fuse
        class="h-80" @update:open="emit('close')"
        @update:model-value="(item) => (selected = true, emit('close', item.puzzle.id))">
        <template #puzzles-label="{ item }">
          <span :class="['puzzle', item.puzzle.meta && 'meta']"
            :style="item.puzzle.meta && `--round-hue: ${item.round.hue}`">
            {{ item.label }}
          </span>
        </template>
      </UCommandPalette>
    </template>
  </UModal>
</template>

<style scoped>
.puzzle {
  font-size: 15px;
  color: oklch(33% 0 0deg);
  font-weight: 400;
}

.meta {
  background:
    linear-gradient(68deg,
      oklch(50% 0.24 var(--round-hue)) 0%,
      oklch(50% 0.24 calc(var(--round-hue) + 60)) 20%,
      oklch(50% 0.24 calc(var(--round-hue) + 180)) 100%);
  background-clip: text;
  color: transparent;
  font-weight: 550;
}
</style>
