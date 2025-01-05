<script setup lang="ts">
const props = defineProps<{
  connected: boolean;
}>();
const config = useAppConfig();

const [discordBase, discordTarget] = useDiscordBase();
const discordURL = computed(() => `${discordBase}/channels/${config.discordGuild}`);
</script>

<template>
  <header>
    <div class="flex-spacer"></div>
    <section>
      <div class="row">
        <ETooltip text="emojihunt / samplepassword" placement="bottom"
          :offset-distance="4">
          <NuxtLink :to="config.huntURL" target="_blank" class="hunt">
            Mystery Hunt 2025
          </NuxtLink>
        </ETooltip>
        <p class="dot"></p>
        <ETooltip text="Big Logistics Email" placement="bottom" :offset-distance="8">
          <NuxtLink to="#" target="_blank">
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
          <NuxtLink to="/">
            <UIcon name="i-heroicons-arrow-right-start-on-rectangle" size="16px" />
          </NuxtLink>
        </ETooltip>
        <p class="dot"></p>
        <span class="emoji">🌊🎨🎡</span>
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
  font-size: 0.85rem;
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

/* Animation */
.ably span {
  /* https://stackoverflow.com/a/16344389 */
  animation: blink 1.5s step-start infinite;
  animation-delay: 2.5s;
  opacity: 0%;
}
</style>
