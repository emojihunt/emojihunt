<script setup lang="ts">
// This page accesses local storage and calls the /authenticate API (which sets
// a cookie). These things need to happen on the client, so server-side
// rendering is disabled for this page in routeRules.

type LoginResult =
  { status: "not_started"; } |
  { status: "canceled"; } |
  { status: "invalid_code"; } |
  { status: "unknown_member", username: string; } |
  { status: "success"; };

const handleLogin = async (url: URL): Promise<LoginResult> => {
  if (url.searchParams.has("error")) {
    return { status: "canceled" };
  }
  const code = url.searchParams.get("code");
  if (!code) {
    return { status: "not_started" };
  }

  try {
    await useAPI("/authenticate", { code });
    return { status: "success" };
  } catch (e: any) {
    // The /authorize endpoint returns HTTP 403 if the code fails to verify. All
    // other errors should bubble up.
    if (e.statusCode === 403) {
      const { username } = e.data || {};
      if (username) {
        return { status: "unknown_member", username };
      }
      return { status: "invalid_code" };
    }
    throw e;
  }
};

const url = useRequestURL();
const returnURL = url.searchParams.get("state") || "/";

const result = await handleLogin(url);
if (result.status === "success") {
  await navigateTo(returnURL);
}
</script>

<template>
  <Login :returnURL="returnURL">
    <span v-if="result.status === 'canceled'">
      Canceled.
    </span>
    <span v-else-if="result.status === 'invalid_code'">
      Invalid or duplicate login attempt.
    </span>
    <span v-else-if="result.status == 'unknown_member'">
      <b>@{{ result.username }}</b> is not a member of the Discord server.
    </span>
  </Login>
</template>
