<script setup lang="ts">
const props = defineProps<{
  puzzle: Puzzle;
  field: "name" | "location" | "description";
  readonly?: boolean;
  style?: "thick" | "thin";
}>();

const content = ref(props.puzzle[props.field].trim());
const editing = ref(false);
const pending = ref(false);
const div = ref<HTMLDivElement>();

const click = () => !props.readonly && !editing.value && beginEdit();
const blur = () => !props.readonly && editing.value && saveEdit();

const keydown = (e: KeyboardEvent) => {
  if (props.readonly) {
    return;
  } else if (e.key == "Enter") {
    if (editing.value) {
      saveEdit();
    }
    else {
      beginEdit();

      // For key press events, we need to manually move focus into the cell. Click
      // events do this automatically.
      const node = div.value!.childNodes[0];
      const range = document.createRange();
      range.setStart(node, 0);
      range.setEnd(node, node.textContent?.length || 0);
      const selection = window.getSelection();
      selection?.removeAllRanges();
      selection?.addRange(range);
    }
    e.preventDefault();
  } else if (e.key == "Escape") {
    if (editing.value) cancelEdit();
  }
};

const beginEdit = () => {
  editing.value = true;
  div.value!.focus();
};

const cancelEdit = () => {
  editing.value = false;
  div.value!.innerText = content.value || "-";
};

const saveEdit = () => {
  let updated = div.value!.textContent?.trim() || "";
  if (updated == "-") {
    updated = "";
  }
  if (updated != content.value) {
    pending.value = true;
    useAPI(`/puzzles/${props.puzzle.id}`, { [props.field]: content.value })
      .then(() => { pending.value = false; });
  };
  editing.value = false;
  content.value = updated;
  div.value!.innerText = content.value || "-";
};
</script>

<template>
  <div ref="div" class="cell" :class="style" @click="click" @blur="blur" @keydown="keydown"
    :contenteditable="editing ? 'plaintext-only' : 'false'" spellcheck="false" tabindex="0">
    {{ content || (editing ? "" : "-") }}
    <Spinner v-if="pending" />
  </div>
</template>

<style scoped>
/* Theming */
div {
  font-size: 0.9rem;
  line-height: 2em;
}

div:hover {
  white-space: unset;
}

/* Custom Styles */
.thick {
  font-weight: 430;
  color: oklch(25% 0.10 275deg);
}

.thin {
  font-weight: 300;
  font-size: 0.86rem;
}
</style>
