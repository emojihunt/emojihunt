<script setup lang="ts">
const props = defineProps<{
  rounds: RoundStats[];
  observer: IntersectionObserver | undefined;
}>();

// Navigate to anchors without changing the fragment:
const navigate = (e: MouseEvent) => {
  const target = e.target! as HTMLAnchorElement;
  const id = (new URL(target.href)).hash;

  document.querySelector(id)?.scrollIntoView();
  e.preventDefault();

  // IntersectionObserver doesn't fire with scrollIntoView, so fix up the
  // `stuck` classes manually.
  if (props.observer) {
    for (const pill of document.querySelectorAll(".pill")) {
      if (pill.getBoundingClientRect().y < 75) {
        pill.classList.add("stuck");
      } else {
        pill.classList.remove("stuck");
      }
    }
  }
};

onMounted(() => document.location.hash && history.pushState(
  "", document.title, window.location.pathname + window.location.search,
));
</script>

<template>
  <header>
    <section class="rounds" v-if="rounds.length > 2">
      <span v-for="round of rounds">
        <a :href="`#${round.anchor}`" @click="navigate">
          {{ round.emoji }}&#xfe0f;
        </a>
        <label v-if="round.complete">✔</label>
      </span>
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
  gap: 0.5rem;
}

span {
  display: flex;
  flex-direction: column;
  align-items: center;
}

a {
  display: block;
  width: 1.75rem;
  line-height: 1.75rem;
  height: 2rem;
}

label {
  display: block;
  width: 0.75rem;
  height: 0.75rem;
  margin-top: -0.4rem;
}


/* Themeing */
header {
  background-color: oklch(98% 0.01 286deg);
  border-bottom: 1px solid oklch(80% 0.01 286deg);
  filter: drop-shadow(0 1.5rem 1rem oklch(100% 0 0deg));

  user-select: none;
}

span {
  opacity: 60%;
}

a {
  background-color: oklch(98% 0.01 286deg);
  border: 1px solid transparent;
  border-radius: 0.33rem;

  text-align: center;
  text-decoration: none;
}

span:hover {
  opacity: 100%;
}

a:hover {
  border-color: oklch(80% 0.01 286deg);
  filter: drop-shadow(0 1px 2px oklch(95% 0 0deg));
}

label {
  color: oklch(60% 0.01 286deg);
  font-size: 0.6rem;
  text-align: center;

  background-color: oklch(98% 0.01 286deg);
  border-radius: 0.3rem;
  pointer-events: none;

  z-index: 3;
}
</style>
