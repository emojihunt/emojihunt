<script setup lang="ts">
const {
  text,
  delayDuration = 250,
  side = "right",
  sideOffset = 0,
} = defineProps<{
  text: string;
  delayDuration?: number;
  side?: "top" | "bottom" | "right" | "left";
  sideOffset?: number;
}>();
</script>

<template>
  <div>
    <UTooltip :text="text" :delay-duration="delayDuration"
      :content="{ side, sideOffset }">
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
