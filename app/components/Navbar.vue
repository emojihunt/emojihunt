<script setup lang="ts">
const props = defineProps<{ rounds: RoundStats[]; }>();

// Navigate to anchors without changing the fragment:
const navigate = (e: MouseEvent) => {
  const target = e.target! as HTMLAnchorElement;
  const id = (new URL(target.href)).hash;
  document.querySelector(id)?.scrollIntoView();
  e.preventDefault();
};

onMounted(() => document.location.hash && history.pushState(
  "", document.title, window.location.pathname + window.location.search,
));
</script>

<template>
  <header>
    <section class="rounds" v-if="rounds.length > 2">
      <a v-for="round of rounds" :href="`#${round.anchor}`" @click="navigate">
        {{ round.emoji }}&#xfe0f;
      </a>
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
  right: 1.75rem;

  display: flex;
}

/* Themeing */
header {
  background-color: oklch(98% 0.01 286deg);
  border-bottom: 1px solid oklch(80% 0.01 286deg);
  filter: drop-shadow(0 1.5rem 1rem oklch(100% 0 0deg));

  user-select: none;
}

a {
  width: 2rem;
  text-align: center;
  opacity: 70%;
  text-decoration: none;
}

a:hover {
  opacity: 100%;
}
</style>
