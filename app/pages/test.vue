<script setup lang="ts">
const data = await useAPI("/puzzles");
</script>

<template>
  <pre>Test, World! {{ data }}</pre>
</template>
