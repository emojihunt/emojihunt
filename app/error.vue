<script setup lang="ts">
import type { NuxtError } from 'nuxt/app';

const props = defineProps<{ error: NuxtError; }>();
const r = <string>('url' in props.error && props.error.url);

const clear = () => clearError({ redirect: '/' });
</script>

<template>
  <Login v-if="error.statusCode == 401 && r" :return="r" />
  <section v-else>
    <div class="center">
      <h1>
        <span class="emoji">🔥</span>Error {{ error.statusCode }}
      </h1>
      <div class="details">
        <div class="message">{{ error.message }}</div>
        <div v-html="error.stack"></div>
        <button @click="clear">Home</button>
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
  width: auto;
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

.message {
  width: 25rem;
}

button {
  margin: 1.8rem 0;
  padding: 0;
  border: none;
  background-color: inherit;
  text-decoration: underline;
  user-select: none;
}

button:hover {
  opacity: 70%;
  cursor: pointer;
}
</style>
