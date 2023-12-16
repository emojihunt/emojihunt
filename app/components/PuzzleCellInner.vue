<script setup lang="ts">
const props = defineProps<{
  puzzle: Puzzle;
  field: "name" | "answer" | "location" | "description";
  tabindex: number;
  readonly?: boolean;
}>();
const emit = defineEmits<{ (e: 'save', b: boolean): void; }>();

const content = ref(props.puzzle[props.field].trim());
const editing = ref(false);
const span = ref<HTMLSpanElement>();

const click = () => {
  if (props.readonly) {
    // no-op
  } else if (editing.value) {
    // no-op
  } else {
    editing.value = true;
    setTimeout(() => span.value?.focus(), 0);
  }
};
const blur = () => !props.readonly && editing.value && saveEdit();
const keydown = (e: KeyboardEvent) => {
  if (props.readonly) {
    return;
  } else if (editing.value) {
    switch (e.key) {
      case "Enter":
        saveEdit();
        window.getSelection()?.removeAllRanges();
        break;
      case "Escape":
        editing.value = false;
        window.getSelection()?.removeAllRanges();
        break;
    }
    e.stopPropagation(); // don't bubble, skip arrow-key handler
  } else {
    switch (e.key) {
      // For key press events, we need to manually move focus into the cell.
      case "Enter":
        editing.value = true;
        const node = span.value!.childNodes[0];
        const range = document.createRange();
        range.setStart(node, 0);
        range.setEnd(node, node.textContent?.length || 0);
        const selection = window.getSelection();
        selection?.removeAllRanges();
        selection?.addRange(range);
        e.preventDefault();
        break;
    }
  }
};

const saveEdit = () => {
  let updated = span.value!.textContent?.trim() || "";
  if (updated == "-") updated = "";
  if (props.field == "answer") {
    updated = updated.toUpperCase();
    if (!updated) {  // answer cannot be blank
      editing.value = false;
      return;
    }
  }
  if (updated != content.value) {
    emit("save", true);
    useAPI(`/puzzles/${props.puzzle.id}`, { [props.field]: updated })
      .then(() => emit("save", false));
  };
  editing.value = false;
  content.value = updated;
};
</script>

<template>
  <span ref="span" :readonly="readonly" @click="click" @blur="blur" @keydown="keydown"
    :contenteditable="editing ? 'plaintext-only' : 'false'" :tabindex="tabindex"
    spellcheck="false">{{ content || (editing ? "" : "-") }}</span>
</template>

<style scoped>
/* Layout */
span {
  flex-grow: 1;
  line-height: 1.35rem;
  padding: 0.25em 0.33rem;
  overflow: hidden;
}

/* Theming */
span {
  white-space: nowrap;
  text-overflow: ellipsis;
}

span:hover,
span:focus,
span[contenteditable='plaintext-only'] {
  white-space: unset;
}

span[contenteditable='plaintext-only'] {
  background-color: oklch(95% 0.03 275deg);
}

span[readonly] {
  cursor: default;
}

span:focus {
  outline: none;
}
</style>
