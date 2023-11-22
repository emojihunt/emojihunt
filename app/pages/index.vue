<script setup>
const data = await useAPI("/puzzles");
</script>

<template>
  <pre>Hello, World! {{ data }}</pre>
</template>
