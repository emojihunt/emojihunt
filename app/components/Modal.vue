<script setup lang="ts">
const emit = defineEmits<{ (event: "close"): void; }>();

const keydown = (e: KeyboardEvent) => {
  e.stopPropagation();
  if (e.key === "Escape") emit("close");
};
</script>

<template>
  <footer @keydown="keydown">
    <section class="ring-1 ring-inset ring-neutral-300">
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
  z-index: 80;
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

/* fix for capturing pointer events across full width of screen */
footer {
  pointer-events: none;
}

section {
  pointer-events: auto;
}
</style>
