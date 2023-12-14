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
      🌍
    </NuxtLink>
    <NuxtLink :href="spreadsheetURL" :ok="!!spreadsheetURL"
      :tabindex="tabIndex(focused, 1)">
      ✏️&#xfe0f;
    </NuxtLink>
    <NuxtLink :href="discordURL" :ok="!!discordURL" :tabindex="tabIndex(focused, 2)">
      👾
    </NuxtLink>
  </nav>
</template>

<style scoped>
/* Layout */
nav {
  display: flex;
  justify-content: center;
}

a {
  width: 2.05rem;
  line-height: 1.75rem;
  filter: opacity(90%);
}

/* Theming */
a {
  text-align: center;
  text-decoration: none;
  user-select: none;
}

a[ok="false"] {
  filter: grayscale(100%) opacity(50%);
  pointer-events: none;
}

a:hover {
  transform: scale(110%);
  filter: drop-shadow(0 2px 1px oklch(85% 0 0deg));
  /* also clears prior opacity() filter */
}
</style>
