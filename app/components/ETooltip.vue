<script setup lang="ts">
const {
  text,
  delay = 250,
  side = "right",
  offset = 4,
} = defineProps<{
  text: string;
  delay?: number;
  side?: "top" | "bottom" | "right" | "left";
  offset?: number;
}>();
</script>

<template>
  <div>
    <UTooltip :text="text" :delay-duration="delay"
      :content="{ side, sideOffset: offset }">
      <slot></slot>
    </UTooltip>
  </div>
</template>

<style scoped>
div {
  user-select: none;
}

:deep(div[data-popper-placement]) {
  z-index: 8;
}
</style>
