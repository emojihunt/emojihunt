<script setup lang="ts">
const props = defineProps<{ open: boolean; }>();
const contents = ref<HTMLElement>();
defineExpose({ contents });
</script>

<template>
  <footer v-if="open">
    <section class="ring-1 ring-inset ring-gray-300" ref="contents">
      <slot></slot>
    </section>
  </footer>
</template>

<style scoped>
/* Layout */
footer {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;

  display: flex;
  justify-content: center;
  z-index: 20;
}

section {
  margin: 1rem;
  padding: 0.5rem;
}

/* Theming */
section {
  border-radius: 0.6rem;
  background-color: white;
  filter: drop-shadow(0 2.5px 4px oklch(77% 0.0 0deg / 40%));
}
</style>
