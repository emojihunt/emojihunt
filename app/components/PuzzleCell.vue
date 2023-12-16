<script setup lang="ts">
const props = defineProps<{
  puzzle: Puzzle;
  field: "name" | "answer" | "location" | "description";
  tabindex: number;
  readonly?: boolean;
}>();

const content = ref(props.puzzle[props.field].trim());
const editing = ref(false);
const pending = ref(false);
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
  if (updated != content.value) {
    pending.value = true;
    useAPI(`/puzzles/${props.puzzle.id}`, { [props.field]: updated })
      .then(() => { pending.value = false; });
  };
  editing.value = false;
  content.value = updated;
};
</script>

<template>
  <div class="cell">
    <span ref="span" :class="field" :readonly="readonly" @click="click" @blur="blur"
      @keydown="keydown" :contenteditable="editing ? 'plaintext-only' : 'false'"
      :tabindex="tabindex" spellcheck="false">{{ content || (editing ? "" : "-") }}</span>
    <Spinner v-if="pending" />
  </div>
</template>

<style scoped>
/* Layout */
.cell {
  display: flex;
  position: relative;
  overflow: hidden;
}

span {
  flex-grow: 1;
  line-height: 1.5em;
  padding: 0.25em 0.33rem;
  overflow: hidden;
}

.spinner {
  position: absolute;
  right: 0.33rem;
  top: calc(1em - 0.5rem - 2px);
}

/* Theming */
span {
  font-size: 0.9rem;
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

.cell:focus-within {
  outline: auto;
}

/* Custom Styles */
.name {
  font-weight: 430;
  color: oklch(25% 0.10 275deg);
}

.answer {
  font-size: 0.87rem;
  font-family: 'IBM Plex Mono', monospace;
  font-weight: 600;
}

.location,
.description {
  font-weight: 300;
  font-size: 0.86rem;
}
</style>
