<script setup>
const { data, error } = useFetch("/api/puzzles");
if (error) throw createError({ statusCode: 403 });
</script>

<template>
  <pre>Test, World! {{ data }}</pre>
</template>
