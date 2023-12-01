<script setup lang="ts">
const config = useAppConfig();
const props = defineProps<{ puzzle: Puzzle; round: RoundStats; }>();
const { hue } = props.round;

const puzzleURL = props.puzzle.puzzle_url.length > 1 ?
  props.puzzle.puzzle_url : '';
const spreadsheetURL = props.puzzle.spreadsheet_id.length > 1 ?
  `https://docs.google.com/spreadsheets/d/${props.puzzle.spreadsheet_id}` : '';
const discordURL = props.puzzle.discord_channel.length > 1 ?
  `https://discord.com/channels/${config.discordGuild}/${props.puzzle.discord_channel}` : '';
</script>

<template>
  <span class="row">
    <div class="buttons">
      <NuxtLink :href="puzzleURL" :ok="!!puzzleURL">
        🌎
      </NuxtLink>
      <NuxtLink :href="spreadsheetURL" :ok="!!spreadsheetURL">
        ✏️&#xfe0f;
      </NuxtLink>
      <NuxtLink :href="discordURL" :ok="!!discordURL">
        💬
      </NuxtLink>
    </div>
    <span class="text">
      <div class="cell name">
        {{ puzzle.name }}
      </div>
      <div class="cell status" v-bind:class="puzzle.answer ? 'solved' : 'unsolved'">
        <span>
          {{ (!puzzle.answer && puzzle.status) ? '✍️' : '' }}
          {{ puzzle.answer || puzzle.status || 'Not Started' }}
        </span>
      </div>
      <div class="cell location">{{ puzzle.location || '-' }}</div>
      <div class="cell note">{{ puzzle.description || '-' }}</div>
    </span>
  </span>
</template>

<style scoped>
/* Layout */
.row {
  grid-column: 1 / 6;
  display: grid;
  grid-template-columns: subgrid;
}

.row div {
  line-height: 1.75em;
}

.text {
  grid-column: 2 / 6;
  display: grid;
  grid-template-columns: subgrid;

  padding-top: 2px;
}

.text div:hover {
  white-space: unset;
}

.buttons {
  display: flex;
  gap: 0.25rem;
  justify-content: center;
}

.buttons a {
  width: 1.85rem;
  height: 1.85rem;
  line-height: 1.85rem;
}

/* Themeing */
.text {
  font-size: 0.9rem;
  border-top: 1px solid transparent;
  border-bottom: 1px solid transparent;
}

.row:hover .text {
  border-color: oklch(86% 0 0deg);
}

.buttons a {
  text-align: center;
  text-decoration: none;

  background-color: white;
  border-radius: 0.33rem;
  user-select: none;
}

.buttons a:hover {
  box-shadow: inset 0 0 2px oklch(90% 0 0deg);
  filter: drop-shadow(0 1px 1px oklch(80% 0 0deg));
}

.buttons a[ok="false"] {
  opacity: 50%;
  filter: grayscale(100%);
  pointer-events: none;
}

.name {
  font-weight: 430;
  color: oklch(23% 0.16 calc(v-bind(hue) + 40));
}

.status {
  cursor: pointer;
  box-sizing: border-box;
  border: 1px solid transparent;
  border-radius: 1px;

  font-size: 0.87rem;
}

.solved {
  font-family: 'IBM Plex Mono', monospace;
}

.unsolved span {
  /* https://stackoverflow.com/a/64127605 */
  margin: 0 -0.4em;
  padding: 0.1em 1.0em 0.1em 0.8em;
  border-radius: 0.75em 0.3em;
  background-image: linear-gradient(90deg,
      oklch(85% 0.25 100deg / 10%),
      oklch(91% 0.19 100deg / 70%) 4%,
      oklch(91% 0.25 100deg / 30%) 92%,
      oklch(91% 0.25 100deg / 0%));
}

.location,
.note {
  font-weight: 300;
  font-size: 0.86rem;
}
</style>
