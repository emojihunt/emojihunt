<script setup lang="ts">
const props = defineProps<{ puzzle: Puzzle; }>();
</script>

<template>
  <fieldset>
    <button v-for="status of Object.values(Status).filter((s) => s != puzzle.status)">
      <span v-if="StatusEmoji(status)" class="emoji" :class="status">
        {{ StatusEmoji(status) }}&#xfe0f;
      </span>
      {{ StatusLabel(status) }}
    </button>
  </fieldset>
</template>

<style scoped>
/* Layout */
fieldset {
  margin: 0 0 0.2rem;
  padding: 0 0.33rem;
  line-height: 1.5em;
}

fieldset button {
  padding: 0.1rem 0.4rem;
  margin: 0.15rem 0.1rem;
}

fieldset .emoji.Purchased {
  padding-right: 1px;
}

/* Theming */
fieldset {
  font-size: 0.8rem;
}

fieldset button {
  border: 1px solid oklch(85% 0 0deg);
  border-radius: 0.6rem;
}

fieldset button:hover,
fieldset button:focus-visible {
  background-color: oklch(95% 0 0deg);
}
</style>
