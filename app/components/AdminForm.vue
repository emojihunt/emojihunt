<script setup lang="ts">
const emit = defineEmits<{ (event: "close"): void; }>();
const toast = useToast();

const data: DiscoveryConfig = await(async () => {
  const { data, error } = await useAPI<DiscoveryConfig>("/discovery");
  if (data.value) return data.value;
  else throw error.value;
})();

let previous: string | number;
const saving = ref(false);
const testing = ref<boolean | Map<string, ScrapedPuzzle[]>>(false);
const submit = async (e: Event) => {
  e.preventDefault();
  saving.value = true;
  if (previous) toast.remove(previous);
  const response = await formSubmit("/discovery", data);
  if (response.status === 401) {
    window.location.reload();
  } else if (response.status === 200) {
    previous = toast.add({
      title: "Updated configuration", color: "success",
      icon: "i-heroicons-check-badge",
    }).id;
    emit("close");
  } else {
    previous = toast.add({
      title: "Error", color: "error", description: (await response.json()).message,
      icon: "i-heroicons-exclamation-triangle",
    }).id;
  }
  saving.value = false;
};
const test = async (e: Event) => {
  e.preventDefault();
  testing.value = true;
  if (previous) toast.remove(previous);
  const response = await formSubmit("/discovery/test", data);
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
      title: "Error", color: "error", description: (await response.json()).message,
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
    <UCheckbox v-model="data.group_mode" label="Group Mode" icon="i-heroicons-check" />
    <UInput v-model="data.cookie_name" placeholder="Cookie Name" />
    <UInput v-model="data.cookie_value" placeholder="Cookie Value" />
    <UInput v-model="data.group_selector" placeholder="Group Selector" />
    <UInput v-model="data.round_name_selector" placeholder="Round Name Selector" />
    <UInput v-model="data.puzzle_list_selector" placeholder="Puzzle List Selector" />
    <UInput v-model="data.puzzle_item_selector" placeholder="Puzzle Item Selector" />
    <UInput v-model="data.websocket_url" placeholder="WebSocket URL" />
    <UInput v-model="data.websocket_token" placeholder="WebSocket Token" />
    <UInput v-model="data.hunt_name" placeholder="Hunt Name" />
    <UInput v-model="data.hunt_url" placeholder="Hunt URL" />
    <UInput v-model="data.hunt_credentials" placeholder="Hunt Credentials" />
    <UInput v-model="data.logistics_url" placeholder="Logistics Email URL" />
    <fieldset>
      <div class="flex-spacer"></div>
      <UButton variant="ghost" type="submit" class="test" @click="test"
        :disabled="saving || testing === true || !data.puzzles_url">
        <Spinner v-if="testing === true" />
        <span v-else>Test</span>
      </UButton>
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
          <NuxtLink :to="puzzle.puzzle_url">{{ puzzle.name }}</NuxtLink>
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

form {
  --form-hue: 150deg;
}

li {
  list-style: circle inside;
  margin: 0 0.5rem;
}

a {
  text-decoration: underline;
}
</style>
