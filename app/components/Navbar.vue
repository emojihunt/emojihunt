<script setup lang="ts">
const props = defineProps<{ rounds: RoundStats[]; }>();
</script>

<template>
  <header>
    <section class="rounds">
      <NuxtLink v-for="round of rounds" v-if="rounds.length > 2" :href="`#${round.anchor}`">
        {{ round.emoji }}&#xfe0f;
      </NuxtLink>
    </section>
  </header>
</template>

<style scoped>
/* Layout */
header {
  width: 100%;
  height: 6rem;
  position: fixed;
}

.rounds {
  position: absolute;
  top: 1rem;
  right: 2rem;

  display: flex;
  gap: 0.5rem;
}

/* Themeing */
header {
  background-color: oklch(98% 0.01 286deg);
  border-bottom: 1px solid oklch(80% 0.01 286deg);
  filter: drop-shadow(0 1.5rem 1rem oklch(100% 0 0deg));

  user-select: none;
}

a {
  opacity: 70%;
  text-decoration: none;
}

a:hover {
  opacity: 100%;
}
</style>
