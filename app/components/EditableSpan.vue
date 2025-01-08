<script setup lang="ts">
const props = withDefaults(defineProps<{
  value: string;
  tabsequence: number;
  placeholder?: string;
  readonly?: boolean;
  sticky?: boolean;
}>(), { placeholder: "-" });

const emit = defineEmits<{
  (event: "save", updated: string): void;
  (event: "cancel"): void;
}>();

const editing = ref(false);
const span = useTemplateRef("span");

// Vue doesn't properly apply reactive updates because it can't track the
// changing state of the contenteditable. Instead, have Vue render the component
// once and control all further updates manually.
const rerender = () => {
  if (!span.value) return;
  span.value.contentEditable = editing.value ? "true" : "false";
  let updated = props.value.trim();
  if (!editing.value && !updated) {
    updated = props.placeholder;
    span.value.classList.add("placeholder");
  } else {
    span.value.classList.remove("placeholder");
  }
  if (span.value.innerText !== updated) {
    span.value.innerText = updated;
  }
};
onMounted(() => rerender());
watch(() => [props.value, props.placeholder], () => {
  if (editing.value) console.log("Props update!",
    props.value, span.value?.innerText);
  editing.value = false;
  rerender();
});

defineExpose({
  focus(): void {
    if (!editing.value) {
      editing.value = true;
      rerender();
    }
    nextTick(() => span.value?.focus());
    props.value && nextTick(() => span.value && highlightContents(span.value));
  },
});

const saveEdit = (): boolean => {
  let updated = span.value?.textContent?.trim() || "";
  if (updated === "-" || updated === props.placeholder) updated = "";
  if (updated !== props.value.trim()) {
    editing.value = false;
    emit("save", updated);
    nextTick(() => rerender());
    return true;
  } else if (!props.sticky) {
    editing.value = false;
    nextTick(() => rerender());
    return true;
  } else {
    return false;
  }
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
        if (saveEdit()) window.getSelection()?.removeAllRanges();
        else e.preventDefault();
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
        if (span.value) highlightContents(span.value);
        rerender();
        e.preventDefault();
        break;
    }
  }
};
</script>

<template>
  <span v-once ref="span" :readonly="readonly" :data-tabsequence="tabsequence"
    @click="click" @blur="blur" @keydown="keydown" spellcheck="false"
    :class="!value && 'placeholder'">{{ value ||
      placeholder }}</span>
</template>

<style scoped>
/* Layout */
span {
  flex-grow: 1;
  line-height: 22px;
  padding: 3px 0.33rem;
  overflow: hidden;
}

/* Theming */
span {
  white-space: nowrap;
  text-overflow: ellipsis;
}

span:hover,
span:focus,
span[contenteditable="true"] {
  white-space: unset;
}

span[contenteditable="true"] {
  background-color: oklch(97% 0.02 275deg);
}

span[readonly] {
  cursor: default;
}

span:focus {
  outline: none;
}

.placeholder {
  color: oklch(60% 0 0deg);
}
</style>
