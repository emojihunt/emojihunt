<script setup lang="ts">
import type { NuxtError } from 'nuxt/app';

const props = defineProps<{ error: NuxtError; }>();
const stack = import.meta.dev && props.error.stack;
if (props.error.statusCode === 401) {
  navigateTo("/login");
} else {
  console.error(props.error);
}
</script>

<template>
  <main>
    <div class="center">
      <h1>
        <span class="emoji">🔥</span>Site Error
      </h1>
      <div class="details">
        <div class="message">{{ error.message }}</div>
        <pre v-if="stack">{{ stack }}</pre>
        <NuxtLink to="/">Return to Home</NuxtLink>
      </div>
    </div>
  </main>
</template>

<style scoped>
main {
  text-align: center;
}

.center {
  text-align: start;
  display: inline-block;
  min-width: 25rem;
  margin: 30vh 0;
}

h1 {
  margin: 0.8rem 0;
  font-size: 1.35rem;
}

.emoji {
  display: inline-block;
  width: 2.2rem;
  user-select: none;
}

.details {
  padding-left: 2.2rem;
}

a {
  display: block;
  margin: 1.8rem 0;

  font-weight: 450;
  color: oklch(58% 0.20 49deg);
  user-select: none;
}

a:hover {
  text-decoration: underline;
  filter: brightness(75%);
}
</style>
