<script setup lang="ts" generic="T">
const { options } = defineProps<{ options: { id: T, label: string; }[]; }>();
const emit = defineEmits<{ (e: "select", id: T): void; }>();
</script>

<template>
  <fieldset>
    <button v-for="option of options" @click="() => emit('select', option.id)">
      {{ option.label }}
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

/* Theming */
fieldset {
  font-size: 0.8125rem;
}

fieldset button {
  border: 1px solid oklch(85% 0 0deg);
  border-radius: 0.6rem;
  outline-offset: 1px;
}

fieldset button:hover {
  background-color: oklch(95% 0 0deg);
}
</style>
