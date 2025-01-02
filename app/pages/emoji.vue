<script setup lang="ts">
// Current: Unicode 15.1
// Update URL: https://github.com/googlefonts/emoji-metadata
import data from "~/assets/emoji-metadata.json";

const computers = ref();
const click = () => {
  if (!computers.value) return;
  const lines = ["["];
  for (const { result } of computers.value) {
    if (!result.length || result[0] < 5 || result[0] > 355) continue;
    lines.push(`  ${JSON.stringify(result).trim()},`);
  }
  lines.push("]");
  console.log("Output", lines.join("\n"));
};
</script>

<template>
  <main @click="click">
    <template v-for="group of data">
      <EmojiColorComputer ref="computers" v-for="emoji of group.emoji"
        :emoji="String.fromCodePoint(...emoji.base)"
        :alternates="emoji.alternates.map(a => String.fromCodePoint(...a))" />
    </template>
    <div class="spacer"></div>
  </main>
</template>

<style scoped>
main {
  display: flex;
  flex-wrap: wrap;
}

main div {
  flex-grow: 1;
}

main div.spacer {
  flex-grow: 1000000000;
}
</style>
