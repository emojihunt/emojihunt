<script setup lang="ts">
const props = defineProps<{
  round: RoundStats;
  selected: boolean;
  observerFixup: () => void;
}>();

const hue = computed(() => props.round.hue);

// Navigate to anchors without changing the fragment
const click = (e: MouseEvent) => {
  const current = useRequestURL();
  const id = (new URL(`#${props.round.anchor}`, current)).hash;
  document.querySelector(id)?.scrollIntoView();
  // @ts-ignore
  document.querySelector(`${id} ~ .row [tabIndex='0']`)?.focus();
  e.preventDefault();
  props.observerFixup();
};
</script>

<template>
  <a :href="`#${round.anchor}`" @click="click" :tabindex="selected ? 0 : -1"
    :aria-label="`To ${round.name}`">
    <span>{{ round.emoji }}&#xfe0f;</span>
    <label v-if="round.complete">✔</label>
  </a>
</template>

<style scoped>
/* Layout */
a {
  display: flex;
  flex-direction: column;
}

span {
  display: block;
  width: 1.75rem;
  line-height: 1.75rem;
  height: 2rem;
}

label {
  display: block;
  width: 100%;
  height: 0.75rem;
  margin: -0.2rem 0 0.1rem;
  pointer-events: none;
}


/* Themeing */
a {
  border: 1.5px solid transparent;
  border-radius: 3px;
  text-decoration: none;
}

span {
  text-align: center;
}

label {
  color: oklch(90% 0.10 v-bind(hue));
  font-size: 0.6rem;
  text-align: center;

  border-radius: 0.3rem;
  z-index: 3;
}

a:hover,
a:focus-visible {
  border: 1.5px solid oklch(95% 0.10 v-bind(hue) / 90%);
  background-color: oklch(95% 0.10 v-bind(hue) / 50%);
}
</style>
