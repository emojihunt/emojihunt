<script setup lang="ts">
const props = defineProps<{
  value: string;
  readonly?: boolean;
  sticky?: boolean;
  tabindex?: number;
}>();

const emit = defineEmits<{
  (event: "save", updated: string): void;
  (event: "cancel"): void;
}>();

const editing = ref(false);
const span = ref<HTMLSpanElement>();

defineExpose({
  focus(): void {
    if (!editing.value) {
      editing.value = true;
      rerender();
    }
    nextTick(() => span.value?.focus());
  },
});

// Vue doesn't properly apply reactive updates because it can't track the
// changing state of the contenteditable. Instead, have Vue render the component
// once and control all further updates manually.
const rerender = () => {
  // if (!span.value) cn;
  span.value!.contentEditable = editing.value ? "plaintext-only" : "false";
  const updated = props.value.trim() || (editing.value ? "" : "-");
  if (span.value!.innerText != updated) {
    span.value!.innerText = updated;
  }
  span.value!.tabIndex = props.tabindex || 0;
};
onMounted(() => rerender());
onUpdated(() => (editing.value = false, rerender()));

const saveEdit = () => {
  let updated = span.value?.textContent?.trim() || "";
  if (updated === "-") updated = "";
  editing.value = false;
  if (updated != props.value.trim()) {
    emit("save", updated);
  }
  nextTick(() => rerender());
};

// With a click event, the browser automatically inserts the caret at the
// position of the click.
const click = () => !props.readonly && !editing.value &&
  (editing.value = true, rerender(), span.value?.focus());

const blur = () => !props.readonly && !props.sticky && editing.value && saveEdit();

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
        emit("cancel");
        rerender();
        break;
    }
    e.stopPropagation(); // don't bubble, skip arrow-key handler
  } else {
    switch (e.key) {
      case "Enter":
        editing.value = true;
        highlightContents(span.value!);
        rerender();
        e.preventDefault();
        break;
    }
  }
};
</script>

<template>
  <span v-once ref="span" :readonly="readonly" @click="click" @blur="blur"
    @keydown="keydown" spellcheck="false">{{ value || "-" }}</span>
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
