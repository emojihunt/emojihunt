<script setup lang="ts">
const { connected, settings, puzzleCount } = usePuzzles();

const [discordBase, discordTarget] = useDiscordBase();
const discordURL = computed(() => settings.discordGuild && settings.hangingOut ?
  `${discordBase}/channels/${settings.discordGuild}/${settings.hangingOut}` : '');;

const logout = async (e: MouseEvent) => {
  e.preventDefault();
  const res = await formSubmit("/logout", {});
  if (res.status === 200) {
    navigateTo("/", { external: true }); // full-page reload
  } else {
    throw createError({
      fatal: true,
      statusCode: res.status,
      data: res._data,
    });
  }
};

const filter = ref(false);
const color = computed(() =>
  filter.value ? "oklch(78% 0.19 245deg)" : "oklch(92% 0.006 265deg)");
defineExpose({ filter });

const ably = useTemplateRef("ably");
onMounted(() => setTimeout(() => ably.value?.classList.add("ready"), 2500));
</script>

<template>
  <header>
    <div class="flex-spacer"></div>
    <section>
      <div class="row">
        <ETooltip :text="settings.huntCredentials || ''" side="bottom" :offset="8">
          <NuxtLink :to="settings.huntURL" target="_blank" class="hunt">
            {{ settings.huntName || 'Mystery Hunt' }}
          </NuxtLink>
        </ETooltip>
        <p class="dot"></p>
        <ETooltip text="Big Logistics Email" side="bottom" :offset="8">
          <NuxtLink :to="settings.logisticsURL" target="_blank">
            <UIcon name="i-heroicons-question-mark-circle" size="16px" />
          </NuxtLink>
        </ETooltip>
        <p class="dot"></p>
        <ETooltip text="Discord" side="bottom" :offset="8">
          <NuxtLink :to="discordURL" :target="discordTarget">
            <UIcon name="i-heroicons-chat-bubble-left-right" size="16px" />
          </NuxtLink>
        </ETooltip>
        <p class="dot"></p>
        <ETooltip text="Log Out" side="bottom" :offset="8">
          <NuxtLink to="/" @click="logout">
            <UIcon name="i-heroicons-arrow-right-start-on-rectangle" size="16px" />
          </NuxtLink>
        </ETooltip>
        <p class="dot"></p>
        <NuxtLink to="/" :external="true" class="logo">
          <span>🌊🎨🎡</span>
        </NuxtLink>
      </div>
      <div class="row" v-if="puzzleCount >= 21">
        <div class="flex-spacer"></div>
        <UFormField name="toggle" class="toggle" label="Priority">
          <USwitch v-model="filter" />
        </UFormField>
      </div>
      <div class="flex-spacer"></div>
      <div class="ably" ref="ably">
        <ETooltip v-if="!connected" text="Live updates paused. Connecting..." side="left">
          <Icon name="i-heroicons-bolt-slash" />
        </ETooltip>
      </div>
    </section>
  </header>
</template>

<style scoped>
/* Layout */
header {
  width: 100%;
  height: var(--header-height);
  position: fixed;
  z-index: 15;

  padding: 0.75rem max(env(safe-area-inset-right), 1rem) 0.5rem;

  display: flex;
}

section {
  display: flex;
  flex-direction: column;
  gap: 0.33rem;
}

section>div {
  display: flex;
  gap: 0.25rem;
  align-items: center;
}

.row a {
  display: flex;
}

.row a:focus-visible {
  outline: 2px solid white;
  outline-offset: 2px;
  border-radius: 1px;
}

.hunt,
.logo {
  line-height: 16px;
}

p.dot {
  width: 0.75rem;
  text-align: center;
}

.toggle {
  display: flex;
  gap: 0.5rem;
}

:deep(.toggle>div) {
  margin: 0;
  display: flex;
  align-items: center;
}

.toggle :deep(button) {
  position: unset;
  height: 1rem;
}

.toggle :deep(button span) {
  height: calc(1rem - 4px);
  width: calc(1rem - 4px);
  margin: 2px 3px;
}

.ably {
  justify-content: right;
}

/* Theming */
header {
  color: white;
  background-color: oklch(30% 0 0deg);
  border-bottom: calc(var(--header-height-outer) - var(--header-height)) solid oklch(80% 0 0deg);
  filter: drop-shadow(0 1.5rem 1rem oklch(100% 0 0deg));
}

section {
  font-size: 0.875rem;
  line-height: 1.65em;
  text-align: right;
}

p.dot:before {
  content: "\00b7";
}

.hunt {
  font-weight: 550;
}

.logo {
  letter-spacing: 0.166em;
  opacity: 70%;
  cursor: default;
}

.toggle {
  --ui-primary: v-bind(color);
}

.toggle :deep(button) {
  outline-color: v-bind(color);
}

.toggle :deep(label) {
  font-size: 0.8125rem;
  line-height: 1.65em;
  font-weight: 450;
  color: v-bind(color);

  cursor: pointer;
  user-select: none;
}

.toggle :deep(button) {
  transform: rotate(180deg);
}

/* Animation */
.ably span {
  opacity: 0%;
}

.ably.ready span {
  /* https://stackoverflow.com/a/16344389 */
  animation: blink 1.5s step-start infinite;
}
</style>
