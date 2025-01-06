<script setup lang="ts">
const props = defineProps<{
  connected: boolean;
}>();
const store = usePuzzles();

const [discordBase, discordTarget] = useDiscordBase();
const discordURL = computed(() => store.discordGuild && store.hangingOut ?
  `${discordBase}/channels/${store.discordGuild}/${store.hangingOut}` : '');;

const logout = async (e: MouseEvent) => {
  e.preventDefault();
  const res = await fetch("/api/logout", { method: "POST" });
  if (res.status === 200) {
    navigateTo("/", { external: true }); // full-page reload
  } else {
    const data = await res.text();
    throw createError({
      fatal: true,
      statusCode: res.status,
      data,
    });
  }
};

const filter = ref(false);
const color = computed(() =>
  filter.value ? "oklch(78% 0.19 245deg)" : "oklch(92% 0.006 265deg)");
defineExpose({ filter });
</script>

<template>
  <header>
    <div class="flex-spacer"></div>
    <section>
      <div class="row">
        <ETooltip :text="store.huntCredentials || ''" placement="bottom"
          :offset-distance="4">
          <NuxtLink :to="store.huntURL" target="_blank" class="hunt">
            {{ store.huntName || 'Mystery Hunt' }}
          </NuxtLink>
        </ETooltip>
        <p class="dot"></p>
        <ETooltip text="Big Logistics Email" placement="bottom" :offset-distance="8">
          <NuxtLink :to="store.logisticsURL" target="_blank">
            <UIcon name="i-heroicons-question-mark-circle" size="16px" />
          </NuxtLink>
        </ETooltip>
        <p class="dot"></p>
        <ETooltip text="Discord" placement="bottom" :offset-distance="8">
          <NuxtLink :to="discordURL" :target="discordTarget">
            <UIcon name="i-heroicons-chat-bubble-left-right" size="16px" />
          </NuxtLink>
        </ETooltip>
        <p class="dot"></p>
        <ETooltip text="Log Out" placement="bottom" :offset-distance="8">
          <NuxtLink to="/" @click="logout">
            <UIcon name="i-heroicons-arrow-right-start-on-rectangle" size="16px" />
          </NuxtLink>
        </ETooltip>
        <p class="dot"></p>
        <span class="emoji">🌊🎨🎡</span>
      </div>
      <div class="row" v-if="store.puzzleCount >= 21">
        <div class="flex-spacer"></div>
        <UFormGroup name="toggle" class="toggle" label="Unsolved">
          <UToggle v-model="filter" />
        </UFormGroup>
      </div>
      <div class="flex-spacer"></div>
      <div class="ably">
        <ETooltip v-if="!connected" text="Live updates paused. Connecting..."
          placement="left" :offset-distance="4" class="ably">
          <Icon name="i-heroicons-signal-slash" />
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

  padding: 0.75rem 1rem 6px;

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
  outline-offset: 2px;
  outline-color: white !important;
}

p.dot {
  width: 0.75rem;
  text-align: center;
}

.toggle {
  display: flex;
  gap: 0.5rem;
}

:deep(.toggle >div) {
  margin: 0;
  display: flex;
  align-items: center;
}

.toggle :deep(button) {
  position: unset;
  height: 1rem;
  width: calc(2rem + 2px);
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

.emoji {
  letter-spacing: 0.166em;
  opacity: 70%;
  cursor: default;
}

.toggle :deep(label) {
  font-size: 0.8125rem;
  line-height: 1.65em;
  font-weight: 450;
  color: v-bind(color);
  user-select: none;
}

.toggle :deep(button) {
  transform: rotate(180deg);
}

.toggle :deep(.focus-visible\:ring-2:focus-visible) {
  box-shadow: none;
  outline-color: white !important;
}

.toggle :deep(button.bg-primary-500) {
  background-color: oklch(72% 0.19 245deg);
}

/* Animation */
.ably span {
  /* https://stackoverflow.com/a/16344389 */
  animation: blink 1.5s step-start infinite;
  animation-delay: 2.5s;
  opacity: 0%;
}
</style>
