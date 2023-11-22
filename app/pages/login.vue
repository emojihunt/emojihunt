<script setup>
let result, error;

const url = useRequestURL();
const code = url.searchParams.get("code");
const canceled = url.searchParams.has("error");

const state = JSON.parse(url.searchParams.get("state")) || {};
const r = state.r || "/";

if (code) {
  try {
    result = await useAPI("/authenticate", { code });
  } catch (e) {
    if (e.statusCode == 400 || e.statusCode == 401) {
      error = e; // we have ui states for these
    } else {
      throw e;
    }
  }
}
</script>

<template>
  <Login :return="r">
    <span v-if="canceled">
      Canceled.
    </span>
    <span v-else-if="error && error.statusCode == 401">
      <b>@{{ error.data.username }}</b> is not a member of the Discord server.
    </span>
    <span v-else-if="error">
      Invalid or duplicate login attempt.
    </span>
    <span v-else-if="result">
      TODO: finish logging in <b>@{{ result.username }}</b>
    </span>
  </Login>
</template>
