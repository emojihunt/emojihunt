<script setup>
const url = useRequestURL();
const code = url.searchParams.get("code");
const canceled = url.searchParams.has("error");

const state = JSON.parse(url.searchParams.get("state")) || {};
const r = state.r || "/";

const event = useRequestEvent();
const result = code ? await useAuthenticateAPI(event, code) : {};
if (!result.error) {
  await navigateTo(r);
}
</script>

<template>
  <Login :return="r">
    <span v-if="canceled">
      Canceled.
    </span>
    <span v-else-if="result.error == 'unknown_member'">
      <b>@{{ result.username }}</b> is not a member of the Discord server.
    </span>
    <span v-else-if="result.error">
      Invalid or duplicate login attempt.
    </span>
  </Login>
</template>
