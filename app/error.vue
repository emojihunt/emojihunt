<script setup lang="ts">
import type { NuxtError } from 'nuxt/app';

const props = defineProps<{ error: NuxtError; }>();
console.error(props.error);

let r: string | undefined;
if ("url" in props.error && typeof props.error.url === "string") {
  r = props.error.url;
}
</script>

<template>
  <Login v-if="error.statusCode == 401 && r" :return="r" />
  <section v-else>
    <div class="center">
      <h1>
        <span class="emoji">🔥</span>Site Error
      </h1>
      <div class="details">
        <div class="message">{{ error.message }}</div>
        <div v-html="error.stack"></div>
        <a href="/">Home</a>
      </div>
    </div>
  </section>
</template>

<style scoped>
section {
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
  color: black;
  user-select: none;
}

a:hover {
  opacity: 70%;
}
</style>
