<script setup lang="ts">
const data = await useAPI("/puzzles");

// HACK: apply hard-coded colors to rounds for testing
const colors = {
  1: 241, 2: 178, 3: 80, 4: 45,
  5: 255, 6: 19, 7: 69, 8: 205,
  9: 28, 10: 24, 11: 141,
};
for (const puzzle of data.value) {
  const id = puzzle.round.id;
  // @ts-ignore
  puzzle.round.color = colors[id];
}

// Group puzzles by round
const puzzles: { [round: number]: any; } = {};
for (const puzzle of data.value) {
  const id: number = puzzle.round.id;
  puzzles[id] ||= [];
  puzzles[id].push(puzzle);
}
</script>

<template>
  <section>
    <ul>
      <li v-for="puzzle in data">
        <div class="round">{{ puzzle.round.emoji }}&#xfe0f;</div>
        <div class="name">{{ puzzle.name }}</div>
        <div class="status">
          <b v-if="puzzle.answer">{{ puzzle.answer }}</b>
          <span v-else> {{ puzzle.status || "Not Started" }}</span>
        </div>
        <div class="note">{{ puzzle.location }} {{ puzzle.description || '-' }}</div>
      </li>
    </ul>
  </section>
</template>

<style scoped>
section {
  margin: 2em auto;
  width: 75rem;
}

ul {
  padding: 0;
}

li {
  display: flex;
  gap: 1em;
  line-height: 1.25em;
  padding: 0.1em 0;
}

li div {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  flex-shrink: 0;
}

.round {
  width: 2em;
  text-align: center;
}

.name {
  width: 20em;
  font-weight: 500;
}

.status {
  width: 20em;
}

.note {
  flex-shrink: 1;
}
</style>
