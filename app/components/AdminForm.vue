<script setup lang="ts">
import type { ScrapedPuzzle } from '~/utils/types';

const emit = defineEmits<{ (event: "close"): void; }>();
const toast = useToast();

const data: DiscoveryConfig = await (async () => {
  const { data, error } = await useFetch<DiscoveryConfig>("/api/discovery");
  if (error.value) {
    throw createError({
      fatal: true,
      message: error.value.message,
      statusCode: error.value.statusCode,
      data: error.value.data,
    });
  }
  return data.value!;
})();

let previous: string;
const saving = ref(false);
const testing = ref<boolean | Map<string, ScrapedPuzzle[]>>(false);
const submit = async (e: Event) => {
  e.preventDefault();
  saving.value = true;
  if (previous) toast.remove(previous);
  const response = await fetch("/api/discovery", {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
    },
    body: (new URLSearchParams(data as any)).toString(),
  });
  if (response.status === 401) {
    window.location.reload();
  } else if (response.status === 200) {
    previous = toast.add({
      title: "Updated configuration", color: "green",
      icon: "i-heroicons-check-badge",
    }).id;
    emit("close");
  } else {
    previous = toast.add({
      title: "Error", color: "red", description: (await response.json()).message,
      icon: "i-heroicons-exclamation-triangle",
    }).id;
  }
  saving.value = false;
};
const test = async (e: Event) => {
  e.preventDefault();
  testing.value = true;
  if (previous) toast.remove(previous);
  const response = await fetch("/api/discovery/test", {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
    },
    body: (new URLSearchParams(data as any)).toString(),
  });
  if (response.status === 401) {
    window.location.reload();
  } else if (response.status === 200) {
    const result = new Map<string, ScrapedPuzzle[]>();
    for (const scraped of await response.json()) {
      if (!result.has(scraped.round_name)) {
        result.set(scraped.round_name, []);
      }
      result.get(scraped.round_name)!.push(scraped);
    }
    testing.value = result;
  } else {
    previous = toast.add({
      title: "Error", color: "red", description: (await response.json()).message,
      icon: "i-heroicons-exclamation-triangle",
    }).id;
    testing.value = false;
  }
};
</script>

<template>
  <h1>Discovery</h1>
  <form>
    <UInput v-model="data.puzzles_url" placeholder="Puzzles URL" autofocus />
    <UCheckbox v-model="data.group_mode" label="Group Mode" />
    <UInput v-model="data.cookie_name" placeholder="Cookie Name" />
    <UInput v-model="data.cookie_value" placeholder="Cookie Value" />
    <UInput v-model="data.group_selector" placeholder="Group Selector" />
    <UInput v-model="data.round_name_selector" placeholder="Round Name Selector" />
    <UInput v-model="data.puzzle_list_selector" placeholder="Puzzle List Selector" />
    <UInput v-model="data.puzzle_item_selector" placeholder="Puzzle Item Selector" />
    <UInput v-model="data.websocket_url" placeholder="WebSocket URL" />
    <UInput v-model="data.websocket_token" placeholder="WebSocket Token" />
    <fieldset>
      <div class="spacer"></div>
      <button type="submit" class="test" @click="test"
        :disabled="saving || testing === true">
        <Spinner v-if="testing === true" />
        <span v-else>Test</span>
      </button>
      <UButton type="button" :disabled="saving || testing === true" @click="submit">
        <Spinner v-if="saving" />
        <span v-else>Update</span>
      </UButton>
    </fieldset>
  </form>
  <section v-if="testing && testing !== true">
    <template v-for="[round, puzzles] of testing">
      <h3>Round: {{ round }}</h3>
      <ul>
        <li v-for="puzzle of puzzles">
          <a :href="puzzle.puzzle_url">{{ puzzle.name }}</a>
        </li>
      </ul>
    </template>
  </section>
</template>

<style scoped>
/* Layout */
h1 {
  margin: 0.5rem;
}

form {
  display: grid;
  grid-template-columns: 1fr 1fr;
  align-items: center;
  margin: 0 0.5rem;
  gap: 0.5rem;
}

fieldset {
  grid-column: 2;
  display: flex;
  gap: 1rem;
  align-items: center;
  justify-content: space-between;
}

.spacer {
  flex-grow: 1;
}

button {
  width: 4.75rem;
  height: 2rem;

  display: flex;
  justify-content: center;
  align-items: center;
}

button.test {
  width: 3rem;
  padding: 0.25rem;
}

section {
  max-height: 50vh;
  overflow-y: scroll;
}

/* Theming */
h1 {
  font-size: 1rem;
  font-weight: 600;
}

button.test {
  font-weight: 500;
  font-size: 0.9rem;
  color: oklch(65% 0.19 150deg);
}

button.test:hover {
  color: oklch(45% 0.19 150deg);
}

li {
  list-style: circle inside;
  margin: 0 0.5rem;
}

a {
  text-decoration: underline;
}
</style>
