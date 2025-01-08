<script setup lang="ts">
const error = useError();
const stack = import.meta.dev && error.value?.stack;
if (error.value?.statusCode === 401) {
  if ('url' in error.value && error.value.url !== "/") {
    const params = new URLSearchParams();
    params.set("return", error.value.url as string);
    navigateTo(`/login?${params.toString()}`);
  } else {
    navigateTo("/login");
  }
} else {
  console.error(error.value);
}
</script>

<template>
  <main>
    <div class="center">
      <h1>
        <span class="emoji">🔥</span>Site Error
      </h1>
      <div class="details">
        <div class="message">{{ error?.message }}</div>
        <pre v-if="stack">{{ stack }}</pre>
        <div class="link">
          <NuxtLink to="/">Return to Home</NuxtLink>
        </div>
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

.link {
  display: block;
  margin: 1.8rem 0;

  font-weight: 450;
  color: oklch(58% 0.20 49deg);
  user-select: none;
}

a:focus-visible {
  outline-offset: 2px;
  outline-color: oklch(45% 0.15 49deg) !important;
}

a:hover {
  text-decoration: underline;
  filter: brightness(75%);
}
</style>
