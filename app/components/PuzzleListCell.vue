<script setup lang="ts">
const props = defineProps<{
  puzzle: Puzzle;
  field: "name" | "location" | "description";
  hue?: number;
  readonly?: boolean;
  style?: "thick" | "thin";
}>();

const content = ref(props.puzzle[props.field].trim());
const editing = ref(false);
const pending = ref(false);

const click = (e: MouseEvent) => {
  if (props.readonly) return;
  const div = e.target as HTMLDivElement;

  editing.value = true;
  div.contentEditable = "plaintext-only";
  div.focus();
};

const blur = (e: FocusEvent) => {
  if (props.readonly) return;
  const div = e.target as HTMLDivElement;

  const updated = div.textContent?.trim() || "";
  if (updated != content.value) {
    content.value = updated;
    pending.value = true;
    useAPI(`/puzzles/${props.puzzle.id}`, { [props.field]: content.value })
      .then(() => { pending.value = false; });
  };
  editing.value = false;
  div.contentEditable = "false";
};
</script>

<template>
  <div class="cell" :class="style" @click="click" @blur="blur" spellcheck="false">
    {{ content || (editing ? "" : "-") }}
    <Spinner v-if="pending" />
  </div>
</template>

<style scoped>
/* Layout */
div {
  line-height: 1.75em;
}

/* Theming */
div {
  font-size: 0.9rem;
}

div:hover {
  white-space: unset;
}

/* Custom Styles */
.thick {
  font-weight: 430;
  color: oklch(33% 0.16 calc(v-bind(hue) + 60));
}

.thin {
  font-weight: 300;
  font-size: 0.86rem;
}
</style>
