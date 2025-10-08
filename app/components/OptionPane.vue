<script setup lang="ts" generic="T">
const { items } = defineProps<{
  items: { id: T, emoji: string; name: string; }[];
}>();
const emit = defineEmits<{ (e: "select", id: T): void; }>();
</script>

<template>
  <fieldset>
    <button v-for="{ id, emoji, name } of items" @click="() => emit('select', id)">
      <span class="emoji" v-if="emoji">{{ emoji }}</span>
      {{ name }}
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

.emoji {
  display: inline-block;
}

/* Theming */
fieldset {
  font-size: 0.8125rem;
}

fieldset button {
  border: 1px solid oklch(85% 0 0deg);
  border-radius: 0.375rem;
  outline-offset: -1px;
}

fieldset button:hover {
  background-color: oklch(95% 0 0deg);
}

fieldset button:hover .emoji {
  transform: scale(110%);
}
</style>
