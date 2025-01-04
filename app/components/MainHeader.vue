<script setup lang="ts">
const props = defineProps<{
  connected: boolean;
}>();
</script>

<template>
  <header>
    <div class="flex-spacer"></div>
    <UTooltip class="ably" text="Live updates paused. Connecting..." :open-delay="250"
      :popper="{ placement: 'auto-end', offsetDistance: 0 }" v-if="!connected">
      <Icon name="i-heroicons-signal-slash" />
    </UTooltip>
  </header>
</template>

<style scoped>
/* Layout */
header {
  width: 100%;
  height: var(--header-height);
  position: fixed;
  z-index: 15;

  padding: 0.75rem 1rem 0;

  display: flex;
  align-items: flex-start;
}

.ably {
  position: absolute;
  bottom: 5px;
  right: 1rem;
  padding: 0 0.5rem;
}

/* Theming */
header {
  color: white;
  background-color: oklch(30% 0 0deg);
  border-bottom: calc(var(--header-height-outer) - var(--header-height)) solid oklch(80% 0 0deg);
  filter: drop-shadow(0 1.5rem 1rem oklch(100% 0 0deg));
  user-select: none;
}

/* Animation */
.ably span {
  /* https://stackoverflow.com/a/16344389 */
  animation: blink 1.5s step-start infinite;
  animation-delay: 2.5s;
  opacity: 0%;
}
</style>
