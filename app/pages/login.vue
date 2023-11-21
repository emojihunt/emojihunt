<script setup>
const url = useRequestURL();
const code = url.searchParams.get("code");
const canceled = !!url.searchParams.get("error");

const state = JSON.parse(url.searchParams.get("state")) || {};
const r = state.r || "/";
</script>

<template>
  <Login :return="r" v-if="canceled">Canceled.</Login>
  <Login :return="r" v-else-if="code">
    <b>@username</b> is not a member of the Discord server.
  </Login>
  <Login :return="r" v-else />
</template>
