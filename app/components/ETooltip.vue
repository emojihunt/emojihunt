<script setup lang="ts">
import type { Placement, PositioningStrategy } from '@popperjs/core';

const {
  text,
  openDelay = 250,
  placement = "right",
  offsetDistance = 0,
  strategy = "fixed" } = defineProps<{
    text: string;
    openDelay?: number;
    placement?: Placement;
    offsetDistance?: number;
    strategy?: PositioningStrategy;
  }>();
</script>

<template>
  <UTooltip :text="text" :open-delay="openDelay"
    :popper="{ placement, offsetDistance, strategy }">
    <slot></slot>
  </UTooltip>
</template>

<style scoped>
:deep(div[data-popper-placement]) {
  z-index: 8;
}
</style>
