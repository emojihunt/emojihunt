<script setup lang="ts">
type Puzzle = {
  id: number,
  name: string,
  answer: string,
  round: Round,
  status: "" | "Working" | "Abandoned" | "Solved" | "Backsolved",
  description: string,
  location: string,
  puzzle_url: string,
  spreadsheet_id: string,
  discord_channel: string,
  original_url: string,
  name_override: string,
  archived: boolean,
  voice_room: string,
};

type Round = {
  id: number,
  name: string,
  emoji: string,
};

const { data: puzzles } = <{ data: Ref<Puzzle[]>; }>await useFetch("http://localhost:8000/puzzles");
const { data: rounds } = <{ data: Ref<Round[]>; }>await useFetch("http://localhost:8000/rounds");
</script>

<template>
  <h1>Rounds</h1>
  <ol>
    <li v-for="round in rounds">{{ round.emoji }} {{ round.name }}</li>
  </ol>

  <h1>Puzzles</h1>
  <ol>
    <li v-for="puzzle in puzzles">
      {{ puzzle.round.emoji }}
      <a :href="puzzle.puzzle_url">
        {{ puzzle.name }}
      </a>
    </li>
  </ol>
</template>
