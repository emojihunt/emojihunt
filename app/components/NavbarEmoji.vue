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
  align-items: center;
}

span {
  display: block;
  width: 1.75rem;
  line-height: 1.75rem;
  height: 2rem;
}

label {
  display: block;
  width: 0.75rem;
  height: 0.75rem;
  margin: -0.33rem 0 0.2rem;
  pointer-events: none;
}


/* Themeing */
a {
  opacity: 60%;
  text-decoration: none;
}

span {
  border: 1.5px solid transparent;
  border-radius: 0.33rem;

  text-align: center;
}

label {
  color: oklch(60% 0 0deg);
  font-size: 0.6rem;
  text-align: center;

  border-radius: 0.3rem;
  z-index: 3;
}

a:hover,
a:focus-visible {
  opacity: 100%;
}

a:hover:not(:focus-visible) span {
  background: oklch(100% 0.10 v-bind(hue) / 33%);
  border-color: oklch(85% 0.10 v-bind(hue));
  filter: drop-shadow(0 1px 2px oklch(95% 0 0deg));
}

a:hover:not(:focus-visible) label {
  background-color: oklch(98% 0 0deg);
}

a:hover label,
a:focus-visible label {
  color: oklch(60% 0.20 v-bind(hue));
}

a:focus-visible {
  background-color: oklch(100% 0.10 v-bind(hue) / 33%);
}
</style>
