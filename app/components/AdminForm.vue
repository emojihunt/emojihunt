<script setup lang="ts">
const emit = defineEmits<{ (event: "close"): void; }>();
const store = usePuzzles();
const toast = useToast();

const data: DiscoveryConfig = await(async () => {
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
    toast.add({
      title: "Updated configuration", color: "green",
      icon: "i-heroicons-check-badge",
    });
  } else {
    toast.add({
      title: "Error", color: "red", description: await response.text(),
      icon: "i-heroicons-exclamation-triangle",
    });
  }
  saving.value = false;
  emit("close");
};
</script>

<template>
  <h1>Discovery</h1>
  <form @submit="submit">
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
      <button type="button" class="test">Test</button>
      <UButton type="submit" :disabled="saving">
        <Spinner v-if="saving" />
        <span v-else>Update</span>
      </UButton>
    </fieldset>
  </form>
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

button.test {
  padding: 0.25rem;
}

/* Theming */
h1 {
  font-size: 1rem;
  font-weight: 600;
}

button {
  justify-content: center;
}

button.test {
  font-weight: 500;
  font-size: 0.9rem;
  color: oklch(65% 0.19 150deg);
}

button.test:hover {
  color: oklch(45% 0.19 150deg);
}
</style>
