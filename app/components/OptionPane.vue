<script setup lang="ts" generic="T">
const { items } = defineProps<{
  items: { id: T, emoji: string; name: string; right: boolean; }[];
  double?: boolean;
}>();
const emit = defineEmits<{ (e: "select", id: T): void; }>();
</script>

<template>
  <fieldset :class="double && 'double'">
    <div class="left">
      <template v-for="{ id, emoji, name, right } of items">
        <button v-if="!right" @click="() => emit('select', id)">
          <span class="emoji">{{ emoji }}</span> {{ name }}
        </button>
      </template>
    </div>
    <div class="right">
      <template v-for="{ id, emoji, name, right } of items">
        <button v-if="right" @click="() => emit('select', id)">
          <span class="emoji">{{ emoji }}</span> {{ name }}
        </button>
      </template>
    </div>
  </fieldset>
</template>

<style scoped>
/* Layout */
fieldset {
  display: grid;
  grid-template-columns: 1fr;
  gap: 0.25rem;

  margin: 0.25rem 0;
  padding: 0 0.33rem;
  line-height: 1.5em;
}

fieldset.double {
  grid-template-columns: 1fr 1fr;
}


@media (max-width: 768px) {
  fieldset.double {
    grid-template-columns: 1fr;
  }
}

fieldset button {
  display: grid;
  grid-template-columns: 16px 1fr;
  gap: 8px;

  width: 100%;
  padding: 2px 6px;
  margin: 0.33rem 0.1rem;
}

.emoji {
  display: inline-block;
}

.wide {
  grid-column: 1 / 3;
}

/* Theming */
fieldset {
  font-size: 0.8125rem;
}

fieldset button {
  border: 1px solid oklch(66% 0 0deg);
  border-radius: 3px;
  outline-offset: 1px;
}

fieldset button:hover {
  background-color: oklch(95% 0 0deg);
}

fieldset button:hover .emoji {
  transform: scale(110%);
}

button {
  text-align: left;
}

.emoji {
  text-align: center;
}
</style>
