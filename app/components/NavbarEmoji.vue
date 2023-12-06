<script setup lang="ts">
const props = defineProps<{
  round: RoundStats;
  navigate: (e: MouseEvent) => void;
}>();

const hue = computed(() => props.round.hue);
</script>

<template>
  <span>
    <a :href="`#${round.anchor}`" @click="navigate">
      {{ round.emoji }}&#xfe0f;
    </a>
    <label v-if="round.complete">✔</label>
  </span>
</template>

<style scoped>
/* Layout */
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
  margin-top: -0.33rem;
}


/* Themeing */
span {
  opacity: 60%;
  pointer-events: none;
}

a {
  border: 1px solid transparent;
  border-radius: 0.33rem;

  text-align: center;
  text-decoration: none;

  pointer-events: auto;
}

span:hover {
  opacity: 100%;
}

span:hover a {
  background: oklch(100% 0.10 v-bind(hue) / 33%);
  border-color: oklch(85% 0.10 v-bind(hue));
  filter: drop-shadow(0 1px 2px oklch(95% 0 0deg));
}

label {
  color: oklch(60% 0 0deg);
  font-size: 0.6rem;
  text-align: center;

  background-color: oklch(98% 0 0deg);
  border-radius: 0.3rem;

  z-index: 3;
}

span:hover label {
  color: oklch(60% 0.20 v-bind(hue))
}
</style>
