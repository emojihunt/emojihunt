<script setup lang="ts">
import { appendResponseHeader } from 'h3';

const event = useRequestEvent();
const redirect_uri = useRedirectURI();
const discord = useCookie("discord", {
  secure: true,
  sameSite: 'lax',
  expires: new Date(4070908800000),
});

const url = useRequestURL();
const code = url.searchParams.get("code");
const state = url.searchParams.get("state");

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
  const { data, error } = await useFetch("/api/authenticate", {
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
    discord.value = state;
    await navigateTo("/"); // success!
  } else if (error.value?.statusCode === 403) {
    // The /authenticate endpoint returns HTTP 403 if the code fails to verify.
    // All other errors are hard errors.
    const { username } = await error.value.data;
    if (username) {
      result.value = { status: "unknown_member", username };
    }
    result.value = { status: "invalid_code" };
  } else {
    throw createError({
      fatal: true,
      statusCode: error.value?.statusCode,
      data: error.value?.data,
    });
  }
}
</script>

<template>
  <Login>
    <span v-if="result?.status === 'canceled'">
      Canceled.
    </span>
    <span v-else-if="result?.status === 'invalid_code'">
      Invalid or duplicate login attempt.
    </span>
    <span v-else-if="result?.status === 'unknown_member'">
      <b>@{{ result.username }}</b> is not a member of the Discord server.
    </span>
  </Login>
</template>
