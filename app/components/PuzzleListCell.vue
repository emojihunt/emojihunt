<script setup lang="ts">
const props = defineProps<{
  content: string;
  hue?: number;
  style?: "thick" | "thin";
}>();
</script>

<template>
  <div class="cell" :class="style">
    {{ content || "-" }}
  </div>
</template>

<style scoped>
/* Layout */
div {
  line-height: 1.75em;
}

/* Theming */
div {
  font-size: 0.9rem;
}

div:hover {
  white-space: unset;
}

/* Custom Styles */
.thick {
  font-weight: 430;
  color: oklch(33% 0.16 calc(v-bind(hue) + 60));
}

.thin {
  font-weight: 300;
  font-size: 0.86rem;
}
</style>
