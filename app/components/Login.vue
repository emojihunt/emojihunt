<script setup lang="ts">
// Preload the fonts used in the app, since they're quite large. We have a
// special subset font for the glyphs used on this page; it *isn't* preloaded
// because it's small and gets inlined by the build step.
import inter from "~/assets/InterVariable.woff2";
import plex from "~/assets/IBMPlexMono-Bold-Latin1.woff2";
import noto from "~/assets/Noto-COLRv1.woff2";
useHead({
  title: "Log in",
  // @ts-ignore
  link: [inter, plex, noto].map((href) =>
    ({ rel: "preload", href, as: "font", type: "font/woff2", crossorigin: true })),
});

const config = useAppConfig();
const props = defineProps<{
  returnURL?: string;
}>();

const params = new URLSearchParams();
params.set("client_id", config.clientID);
params.set("redirect_uri", useRedirectURI());
params.set("response_type", "code");
params.set("scope", "identify");
params.set("state", props.returnURL || "/");

const authorize = `oauth2/authorize?${params.toString()}`;
</script>

<template>
  <main>
    <h1>🌊🎨🎡</h1>
    <section>
      <h2>Log in</h2>
      <ul>
        <li>
          <span>📱</span>
          <a :href="`discord:///${authorize}`">via Discord app</a>
          <a href="https://discord.com/download" class="download">
            <UIcon name="i-heroicons-arrow-down-tray-20-solid" />
          </a>
        </li>
        <li>
          <span>🌐</span>
          <a :href="`https://discord.com/${authorize}`">via discord.com</a>
        </li>
      </ul>
    </section>
    <div class="error" v-if="$slots.default">
      <slot></slot>
    </div>
  </main>
</template>

<style scoped>
main {
  padding: 1rem;
  margin: 25vh auto;
  max-width: 26rem;
  font-family: "Inter Variable Login", "Noto Color Emoji Login", sans-serif;

  display: flex;
  flex-direction: column;
  gap: 0.7rem;
}

h1 {
  text-align: right;
  margin: 0 2px 0 0;
  font-size: 1.2rem;
  letter-spacing: 0.2rem;
  user-select: none;
  opacity: 70%;
  filter: drop-shadow(0 2.5px 4px oklch(82% 0.10 243deg / 40%));
}

section {
  padding: 1rem 1.25rem;
  border: 2px solid oklch(40% 0.21 274deg);
  border-radius: 3px;

  display: flex;
  gap: 1rem;

  color: oklch(40% 0.21 274deg);
  user-select: none;
}

h2 {
  font-weight: 650;
}

ul {
  font-weight: 450;
}

li:first-child {
  margin-bottom: 0.7rem;
}

li span {
  padding: 0 0.25rem;
}

.download {
  margin: 0 0.4rem;
  padding: 0.2rem 0.2rem 0;
  border-radius: 2px;
}

a:hover {
  text-decoration: underline;
  filter: brightness(75%);
}

.download:hover {
  filter: none;
  color: white;
  background-color: oklch(33% 0.21 274deg);
}

.error {
  margin-left: calc(1rem + 2px);
  color: oklch(60% 0.20 24deg);
}
</style>
