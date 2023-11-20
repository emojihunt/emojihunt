<script setup lang="ts">
type Config = {
  apiBase: string,
  clientID: string,
};

type Authentication = {
  api_key: string,
  username: string,
};

type Puzzle = {
  id: number,
  name: string,
  answer: string,
  round: Round,
  status: "" | "Working" | "Abandoned" | "Solved" | "Backsolved",
  description: string,
  location: string,
  puzzle_url: string,
  spreadsheet_id: string,
  discord_channel: string,
  original_url: string,
  name_override: string,
  archived: boolean,
  voice_room: string,
};

type Round = {
  id: number,
  name: string,
  emoji: string,
};

const url = useRequestURL();
const config = <Config>useRuntimeConfig().public;

const token = new URLSearchParams(url.hash.substring(1)).get("access_token");
if (!token) {
  const authenticate = new URL("https://discord.com/api/oauth2/authorize");
  const params = authenticate.searchParams;
  params.set("client_id", config.clientID);
  params.set("redirect_url", url.toString());
  params.set("response_type", "token");
  params.set("scope", "identify");
  navigateTo(authenticate.toString(), { external: true });
  throw "Redirecting...";
}

const params = new URLSearchParams();
params.set("access_token", token);
const { data: auth } = await < { data: Ref<Authentication>; } > useFetch(
  config.apiBase + "/authenticate",
  {
    method: "POST",
    body: params.toString(),
    headers: {
      "Content-Type": "application/x-www-form-urlencoded"
    },
  },
);
const { data: puzzles } = <{ data: Ref<Puzzle[]>; }>await useFetch(
  config.apiBase + "/puzzles",
  { headers: { Authorization: `Bearer ${auth.value.api_key}` } },
);
const { data: rounds } = <{ data: Ref<Round[]>; }>await useFetch(
  config.apiBase + "/rounds",
  { headers: { Authorization: `Bearer ${auth.value.api_key}` } },
);
</script>

<template>
  <h1>Hello, {{ auth.username }}!</h1>

  <h1>Rounds</h1>
  <ol>
    <li v-for="round in rounds" :value=round.id>{{ round.emoji }} {{ round.name }}</li>
  </ol>

  <h1>Puzzles</h1>
  <ol>
    <li v-for="puzzle in puzzles" :value=puzzle.id>
      {{ puzzle.round.emoji }}
      <a :href="puzzle.puzzle_url">
        {{ puzzle.name }}
      </a>
    </li>
  </ol>
</template>
