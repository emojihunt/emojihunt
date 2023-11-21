<script setup lang="ts">
import type { NuxtError } from 'nuxt/app';

const props = defineProps<{ error: NuxtError; }>();
const r = <string>('url' in props.error && props.error.url);

const handleError = () => clearError({ redirect: '/' });
</script>

<template>
  <Login v-if="error.statusCode == 403 && r" :return="r" />
  <div v-else>
    <h2>HTTP {{ error.statusCode }}</h2>
    <div>{{ error.message }}</div>
    <button @click="handleError">Clear errors</button>
  </div>
</template>
