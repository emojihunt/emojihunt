<script setup lang="ts">
import { appendResponseHeader } from 'h3';

// Preload the fonts used in the app, since they're quite large. We have a
// special subset font for the glyphs used on this page; it *isn't* preloaded
// because it's small and gets inlined by the build step.
import inter from "~/assets/InterVariable.woff2";
import plex from "~/assets/IBMPlexMono-Bold-Latin1.woff2";
import noto from "~/assets/Noto-COLRv1.woff2";

useHead({
  htmlAttrs: { lang: "en" },
  title: "Puzzle Tracker",
  // @ts-ignore
  link: [inter, plex, noto].map((href, i) =>
    ({ rel: "preload", href, as: "font", type: "font/woff2", crossorigin: true, key: `preload-${i}` })),
});
useSeoMeta({
  description: "Welcome to the 🌊🎨🎡 puzzle tracker! Please log in.",
});

const event = useRequestEvent();
const redirect_uri = useRedirectURI();
const discord = useCookie("discord", {
  secure: true,
  sameSite: 'lax',
  expires: new Date(4070908800000),
});

const url = useRequestURL();
const code = url.searchParams.get("code");
const ret = url.searchParams.get("return");
const state = url.searchParams.get("state");

const { clientID } = useAppConfig();
const params = new URLSearchParams();
params.set("client_id", clientID);
params.set("redirect_uri", useRedirectURI());
params.set("response_type", "code");
params.set("scope", "identify");
params.set("state", ret || "");

const authorize = `oauth2/authorize?${params.toString()}`;

type LoginError =
  { status: "canceled"; } |
  { status: "invalid_code"; } |
  { status: "unknown_member", username: string; };

const result = ref<LoginError>();
if (url.searchParams.has("error")) {
  result.value = { status: "canceled" };
} else if (!code) {
  // not started
} else {
  const { data, error } = await useAPI("/authenticate", {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
    },
    body: (new URLSearchParams({ code, redirect_uri })).toString(),
    onResponse({ response }) {
      const cookie = response.headers.getSetCookie();
      if (import.meta.server && cookie && event) {
        // If this call was made during server-side rendering, make sure to pass
        // through the Set-Cookie header:
        // https://nuxt.com/docs/getting-started/data-fetching#pass-cookies-from-server-side-api-calls-on-ssr-response
        appendResponseHeader(event, 'set-cookie', cookie);
      }
    },
  });

  if (data.value) {
    discord.value = state?.endsWith("A") ? "app" : "web";
    await navigateTo(state?.slice(0, -1) || "/"); // success!
  } else if (error.value?.statusCode === 403) {
    // The /authenticate endpoint returns HTTP 403 if the code fails to verify.
    // All other errors are hard errors.
    const { username } = error.value.data;
    if (username) {
      result.value = { status: "unknown_member", username };
    } else {
      result.value = { status: "invalid_code" };
    }
  } else {
    throw error.value;
  }
}
</script>

<template>
  <main>
    <h1>🌊🎨🎡</h1>
    <section>
      <h2>Log in</h2>
      <ul>
        <li>
          <NuxtLink :to="`discord://app/${authorize}A`">
            📱 <span class="link">via Discord app</span>
          </NuxtLink>
          <NuxtLink to="https://discord.com/download" id="download"
            aria-label="download Discord app">
            <UIcon name="i-heroicons-arrow-down-tray-20-solid" />
          </NuxtLink>
        </li>
        <li>
          <NuxtLink :to="`https://discord.com/${authorize}W`">
            🌐 <span class="link">via discord.com</span>
          </NuxtLink>
        </li>
      </ul>
    </section>
    <div class="error" role="alert" v-if="result?.status === 'canceled'">
      Login canceled.
    </div>
    <div class="error" role="alert" v-else-if="result?.status === 'invalid_code'">
      Error: invalid or duplicate login attempt.
    </div>
    <div class="error" role="alert" v-else-if="result?.status === 'unknown_member'">
      Error: <b>@{{ result.username }}</b> is not a member of the Discord server.
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
  border: 3px solid transparent;
  border-radius: 4px;
  background: linear-gradient(white, white) padding-box,
    linear-gradient(68deg, oklch(33% 0.21 260deg), oklch(50% 0.21 290deg)) border-box;

  display: flex;
  gap: 1rem;

  user-select: none;
}

h2 {
  font-weight: 600;
  color: oklch(40% 0.21 270deg);
}

ul {
  color: oklch(50% 0.21 280deg);
}

li:first-child {
  margin-bottom: 0.7rem;
}

a {
  padding: 0.1rem 0.1rem;
}

a:focus-visible {
  outline-color: oklch(50% 0.15 274deg) !important;
}

a:hover .link {
  text-decoration: underline;
  color: oklch(33% 0.21 280deg);
}

#download {
  margin: 0 0.5rem;
  padding: 0.2rem 0.2rem 0;
}

#download {
  color: oklch(50% 0.21 290deg);
}

#download:hover {
  filter: none;
  color: white;
  background-color: oklch(33% 0.21 290deg);
}

.error {
  margin-left: calc(1rem + 2px);
  color: oklch(60% 0.20 24deg);
}
</style>
