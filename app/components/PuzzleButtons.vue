<script setup lang="ts">
const config = useAppConfig();
const props = defineProps<{ puzzle: Puzzle; focused: FocusInfo; }>();

const puzzleURL = props.puzzle.puzzle_url.length > 1 ?
  props.puzzle.puzzle_url : '';
const spreadsheetURL = props.puzzle.spreadsheet_id.length > 1 ?
  `https://docs.google.com/spreadsheets/d/${props.puzzle.spreadsheet_id}` : '';
const discordURL = props.puzzle.discord_channel.length > 1 ?
  `https://discord.com/channels/${config.discordGuild}/${props.puzzle.discord_channel}` : '';
</script>

<template>
  <nav>
    <NuxtLink :href="puzzleURL" :ok="!!puzzleURL" :tabindex="tabIndex(focused, 0)">
      🌎
    </NuxtLink>
    <NuxtLink :href="spreadsheetURL" :ok="!!spreadsheetURL"
      :tabindex="tabIndex(focused, 1)">
      ✏️&#xfe0f;
    </NuxtLink>
    <NuxtLink :href="discordURL" :ok="!!discordURL" :tabindex="tabIndex(focused, 2)">
      💬
    </NuxtLink>
  </nav>
</template>

<style scoped>
/* Layout */
nav {
  display: flex;
  gap: 0.25rem;
  justify-content: center;
}

a {
  width: 1.75rem;
  height: 1.75rem;
  line-height: 1.75rem;
  margin: 0.05rem 0;
}

/* Theming */
a {
  text-align: center;
  text-decoration: none;

  background-color: white;
  border-radius: 0.33rem;
  user-select: none;
}

a:hover {
  box-shadow: inset 0 0 1px oklch(70% 0 0deg);
  filter: drop-shadow(0 0 1px oklch(85% 0 0deg));
}

a[ok="false"] {
  opacity: 50%;
  filter: grayscale(100%);
  pointer-events: none;
}

a:focus-visible {
  z-index: 1;
}

a:hover {
  z-index: 3;
}
</style>
