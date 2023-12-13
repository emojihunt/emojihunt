<script setup lang="ts">
const config = useAppConfig();
const props = defineProps<{ puzzle: Puzzle; focused: FocusInfo; }>();

const puzzleURL = props.puzzle.puzzle_url.length > 1 ?
  props.puzzle.puzzle_url : '';
const spreadsheetURL = props.puzzle.spreadsheet_id.length > 1 ?
  `https://docs.google.com/spreadsheets/d/${props.puzzle.spreadsheet_id}` : '';
const discordURL = props.puzzle.discord_channel.length > 1 ?
  `https://discord.com/channels/${config.discordGuild}/${props.puzzle.discord_channel}` : '';
</script>

<template>
  <nav>
    <!-- Icons from OpenMoji, https://openmoji.org/. -->
    <NuxtLink :href="puzzleURL" :ok="!!puzzleURL" :tabindex="tabIndex(focused, 0)">
      <svg id="emoji" viewBox="0 0 72 72" xmlns="http://www.w3.org/2000/svg">
        <g id="color">
          <circle cx="36" cy="36" r="28" fill="#92D3F5" stroke="none" stroke-linecap="round"
            stroke-linejoin="round" stroke-miterlimit="10" stroke-width="2" />
          <path fill="#B1CC33" stroke="none" stroke-linecap="round" stroke-linejoin="round"
            stroke-miterlimit="10" stroke-width="2"
            d="M41.6878,29.1965c-0.0568,0.7946-2.2706,3.5222-2.8378,3.9161c-1.1826,0.8214-1.6443,0.4449-2.6746,1.2557 c-0.653,0.5139-1.469,1.0634-2.4689,1.2557c-0.5534,0.1064-0.3086-0.4165-1.5537,0.3831c-0.6936,0.4454-0.0778,0.9247,0,1.3621 c0.1046,0.588,0.1828,0.2336,1.3621,1.3621c0.6157,0.5893,0.6707,1.1144,1.3621,1.3621c0.4322,0.1549,0.7463,0.07,1.3621,0 c1.2911-0.1467,0.717,1.1259,1.3621,1.3621C39.0851,41.9988,39.6331,41.7716,41,42c0.552,0.0922,1.3122-0.1799,2,1 c0.2963,0.5083,0,2,0.05,2.5421c0.0844,0.9159-0.5065,1.7288-1.3621,2.7243c-0.6921,0.8052-0.8206,0.6811-1.3621,1.3621 c-0.7236,0.9099-0.5299,1.1763-1.3621,2.7243c-0.8424,1.5667-1.3055,1.7862-1.3621,2.7243c-0.0237,0.3923-0.0011,1.32,0,1.3621 c0.0238,0.9465,0.0765,1.0716,0,1.3621c-0.1781,0.677-1.5528,1.2699-2.0716,1.1919c-1.0211-0.1535-1.0026-3.6818-1.05-4.2283 c-0.0958-1.1035,0.3564-1.7204,0.3973-2.4121c0.1412-2.393,0.5408-3.1351,0-4.0864c-0.3451-0.607-0.5273-0.3396-1.3621-1.3621 c-0.4715-0.5774-1.1856-1.4733-1.3621-2.7243c-0.1113-0.788,0.1212-0.7928,0-1.3621c-0.2251-1.0562-1.1013-1.3971-2.7243-2.7243 c-1.0013-0.8188-1.8894-1.5449-2.7243-2.7243c-0.4339-0.6128-1.0478-1.48-1.3621-2.7243c-0.4138-1.6378,0.2025-1.9051,0-4.0864 c-0.0457-0.4913-0.2445-2.2829-1.3621-4.0864c-0.4745-0.7657-0.8087-1.2882-1.3621-1.3621c-1.055-0.141-1.6211,1.519-2.7243,1.3621 c-0.6593-0.0938-1.1689-0.788-1.3621-1.3621c-0.0517-0.1534-0.2402-0.7406,0-1.3621c0.1331-0.3445,0.5428-0.6837,1.3621-1.3621 c0.836-0.6924,1.2563-1.045,1.3621-1.3621c0.1992-0.5974-0.1599-0.8105,0-1.3621c0.2735-0.944,1.7229-1.6939,2.7243-1.3621 c0.5752,0.1906,0.4511,0.5555,1.3621,1.3621c0.4423,0.3916,1.3981,1.1259,2.7243,1.3621c0.6697,0.1193,1.0045,0.1788,1.3621,0 c0.8809-0.4408,0.4597-1.6084,1.3621-2.7243c0.8301-1.0264,2.1787-1.2654,2.7243-1.3621c1.3486-0.2391,1.5329,0.2732,2.7243,0 c1.2802-0.2937,1.2433-0.9257,2.7243-1.3621c0.6485-0.1911,1.791-0.528,2.7243,0c0.8401,0.4752,0.7287,1.2093,1.3621,1.3621 c0.884,0.2134,1.3172-1.1646,2.7243-1.3621c1.0743-0.1508,2.3522,0.4375,2.7243,1.3621c0.1847,0.4589,0.0982,0.8817,0,1.3621 c-0.2966,1.4511-1.0836,1.5389-1.3621,2.7243c-0.3028,1.288,0.3458,1.6004,0,2.7243c-0.227,0.7378-0.6811,0.5108-1.4047,1.1067 c-0.3311,0.2727-0.4115,0.7094-1.3196,1.6175c-0.454,0.454-0.9255,0.8914-1.3621,1.3621 C42.0546,28.9072,41.8959,28.9885,41.6878,29.1965" />
        </g>
        <g id="line">
          <circle cx="36" cy="36" r="28" fill="none" stroke="#000000" stroke-linecap="round"
            stroke-linejoin="round" stroke-miterlimit="10" stroke-width="2" />
          <path fill="none" stroke="#000000" stroke-linecap="round" stroke-linejoin="round"
            stroke-miterlimit="10" stroke-width="2"
            d="M41.6878,29.1965c-0.0568,0.7946-2.2706,3.5222-2.8378,3.9161c-1.1826,0.8214-1.6443,0.4449-2.6746,1.2557 c-0.653,0.5139-1.469,1.0634-2.4689,1.2557c-0.5534,0.1064-0.3086-0.4165-1.5537,0.3831c-0.6936,0.4454-0.0778,0.9247,0,1.3621 c0.1046,0.588,0.1828,0.2336,1.3621,1.3621c0.6157,0.5893,0.6707,1.1144,1.3621,1.3621c0.4322,0.1549,0.7463,0.07,1.3621,0 c1.2911-0.1467,0.717,1.1259,1.3621,1.3621C39.0851,41.9988,39.6331,41.7716,41,42c0.552,0.0922,1.3122-0.1799,2,1 c0.2963,0.5083,0,2,0.05,2.5421c0.0844,0.9159-0.5065,1.7288-1.3621,2.7243c-0.6921,0.8052-0.8206,0.6811-1.3621,1.3621 c-0.7236,0.9099-0.5299,1.1763-1.3621,2.7243c-0.8424,1.5667-1.3055,1.7862-1.3621,2.7243c-0.0237,0.3923-0.0011,1.32,0,1.3621 c0.0238,0.9465,0.0765,1.0716,0,1.3621c-0.1781,0.677-1.5528,1.2699-2.0716,1.1919c-1.0211-0.1535-1.0026-3.6818-1.05-4.2283 c-0.0958-1.1035,0.3564-1.7204,0.3973-2.4121c0.1412-2.393,0.5408-3.1351,0-4.0864c-0.3451-0.607-0.5273-0.3396-1.3621-1.3621 c-0.4715-0.5774-1.1856-1.4733-1.3621-2.7243c-0.1113-0.788,0.1212-0.7928,0-1.3621c-0.2251-1.0562-1.1013-1.3971-2.7243-2.7243 c-1.0013-0.8188-1.8894-1.5449-2.7243-2.7243c-0.4339-0.6128-1.0478-1.48-1.3621-2.7243c-0.4138-1.6378,0.2025-1.9051,0-4.0864 c-0.0457-0.4913-0.2445-2.2829-1.3621-4.0864c-0.4745-0.7657-0.8087-1.2882-1.3621-1.3621c-1.055-0.141-1.6211,1.519-2.7243,1.3621 c-0.6593-0.0938-1.1689-0.788-1.3621-1.3621c-0.0517-0.1534-0.2402-0.7406,0-1.3621c0.1331-0.3445,0.5428-0.6837,1.3621-1.3621 c0.836-0.6924,1.2563-1.045,1.3621-1.3621c0.1992-0.5974-0.1599-0.8105,0-1.3621c0.2735-0.944,1.7229-1.6939,2.7243-1.3621 c0.5752,0.1906,0.4511,0.5555,1.3621,1.3621c0.4423,0.3916,1.3981,1.1259,2.7243,1.3621c0.6697,0.1193,1.0045,0.1788,1.3621,0 c0.8809-0.4408,0.4597-1.6084,1.3621-2.7243c0.8301-1.0264,2.1787-1.2654,2.7243-1.3621c1.3486-0.2391,1.5329,0.2732,2.7243,0 c1.2802-0.2937,1.2433-0.9257,2.7243-1.3621c0.6485-0.1911,1.791-0.528,2.7243,0c0.8401,0.4752,0.7287,1.2093,1.3621,1.3621 c0.884,0.2134,1.3172-1.1646,2.7243-1.3621c1.0743-0.1508,2.3522,0.4375,2.7243,1.3621c0.1847,0.4589,0.0982,0.8817,0,1.3621 c-0.2966,1.4511-1.0836,1.5389-1.3621,2.7243c-0.3028,1.288,0.3458,1.6004,0,2.7243c-0.227,0.7378-0.6811,0.5108-1.4047,1.1067 c-0.3311,0.2727-0.4115,0.7094-1.3196,1.6175c-0.454,0.454-0.9255,0.8914-1.3621,1.3621 C42.0546,28.9072,41.8959,28.9885,41.6878,29.1965" />
        </g>
      </svg>
    </NuxtLink>
    <NuxtLink :href="spreadsheetURL" :ok="!!spreadsheetURL"
      :tabindex="tabIndex(focused, 1)">
      <svg id="emoji" viewBox="0 0 72 72" xmlns="http://www.w3.org/2000/svg">
        <g id="color">
          <polygon fill="#3F3F3F" stroke="none"
            points="14.5943,53.154 18.2544,56.814 12.6242,59.0928" />
          <polyline fill="#f4aa41" stroke="none"
            points="18.0381,41.8761 36.8684,23.0457 48.1813,34.3586 29.5108,53.0291" />
          <polyline fill="#a57939" stroke="none"
            points="14.4132,52.5663 17.9281,43.3946 27.8278,53.2939 18.6558,56.809" />
          <polyline fill="#EA5A47" stroke="none"
            points="42.9209,16.9933 50.4228,9.4913 61.7357,20.8042 54.2975,28.2424" />
          <polyline fill="#9b9b9a" stroke="none"
            points="35.6498,24.2643 43.3318,16.5823 54.6447,27.8952 47.0278,35.512" />
        </g>
        <g id="line">
          <polygon fill="none" stroke="#000000" stroke-linecap="round"
            stroke-linejoin="round" stroke-miterlimit="10" stroke-width="2"
            points="18.6304,56.8203 27.8278,53.2939 53.8207,27.301 43.9212,17.4015 17.9281,43.3946 14.3904,52.6032" />
          <polyline fill="none" stroke="#000000" stroke-linecap="round"
            stroke-linejoin="round" stroke-miterlimit="10" stroke-width="2"
            points="47.3354,13.9873 50.8388,10.4839 60.7383,20.3834 57.2645,23.8572" />
          <line x1="18.5562" x2="28.4557" y1="42.7665" y2="52.6659" fill="none"
            stroke="#000000" stroke-linecap="round" stroke-linejoin="round"
            stroke-miterlimit="10" stroke-width="2" />
          <polyline fill="none" stroke="#000000" stroke-linecap="round"
            stroke-linejoin="round" stroke-miterlimit="10" stroke-width="2"
            points="14.3979,52.5817 11.9073,59.3148 18.6558,56.809" />
          <line x1="36.9099" x2="46.4225" y1="25.0073" y2="34.5199" fill="none"
            stroke="#000000" stroke-linecap="round" stroke-linejoin="round"
            stroke-miterlimit="10" stroke-width="2" />
        </g>
      </svg>

    </NuxtLink>
    <NuxtLink :href="discordURL" :ok="!!discordURL" :tabindex="tabIndex(focused, 2)">
      <svg id="emoji" viewBox="0 0 72 72" xmlns="http://www.w3.org/2000/svg">
        <g id="color">
          <polygon fill="#8967AA"
            points="10,21 10,35 13,35 13,38 16,38 16,45 22,45 22,51 27,51 28,51 28,45 44,45 44,51 50,51 50,45 56,45 56,38 59,38 59,35 62,35 62,20 56,20 56,24 56,25 49,25 49,20 45,20 45,14 39,14 39,20 32,20 32,14 26,14 26,20 22,20 22,25 16,25 16,20 10,20" />
          <rect x="16" y="51" width="6" height="6" fill="#8967AA" />
          <rect x="50" y="51" width="6" height="6" fill="#8967AA" />
        </g>
        <g id="line">
          <rect x="25.175" y="31" width="3.6" height="6" stroke="#000000"
            stroke-linecap="round" stroke-linejoin="round" stroke-width="2" />
          <polyline fill="none" stroke="#000000" stroke-linecap="round"
            stroke-linejoin="round" stroke-width="2" points="22,45 16,45 16,39" />
          <polyline fill="none" stroke="#000000" stroke-linecap="round"
            stroke-linejoin="round" stroke-width="2" points="22.583,25 22.583,20 26,20" />
          <polyline fill="none" stroke="#000000" stroke-linecap="round"
            stroke-linejoin="round" stroke-width="2"
            points="48.708,25 48.708,20 45.292,20" />
          <polyline fill="none" stroke="#000000" stroke-linecap="round"
            stroke-linejoin="round" stroke-width="2"
            points="13,35 10,35 10,20 16,20 16,35" />
          <polyline fill="none" stroke="#000000" stroke-linecap="round"
            stroke-linejoin="round" stroke-width="2"
            points="56,35 56,20 62,20 62,35 59,35" />
          <polyline fill="none" stroke="#000000" stroke-linecap="round"
            stroke-linejoin="round" stroke-width="2" points="26,20 26,14 32,14 32,20" />
          <polyline fill="none" stroke="#000000" stroke-linecap="round"
            stroke-linejoin="round" stroke-width="2" points="39,20 39,14 45,14 45,20" />
          <polyline fill="none" stroke="#000000" stroke-linecap="round"
            stroke-linejoin="round" stroke-width="2"
            points="16,35 19,35 19,38 13,38 13,35" />
          <polyline fill="none" stroke="#000000" stroke-linecap="round"
            stroke-linejoin="round" stroke-width="2"
            points="59,35 59,38 53,38 53,35 56,35" />
          <rect x="16" y="51" width="6" height="6" fill="none" stroke="#000000"
            stroke-linecap="round" stroke-linejoin="round" stroke-width="2" />
          <rect x="50" y="51" width="6" height="6" fill="none" stroke="#000000"
            stroke-linecap="round" stroke-linejoin="round" stroke-width="2" />
          <polyline fill="none" stroke="#000000" stroke-linecap="round"
            stroke-linejoin="round" stroke-width="2" points="28,45 28,51 22,51 22,45" />
          <polyline fill="none" stroke="#000000" stroke-linecap="round"
            stroke-linejoin="round" stroke-width="2" points="50,45 56,45 56,39" />
          <polyline fill="none" stroke="#000000" stroke-linecap="round"
            stroke-linejoin="round" stroke-width="2" points="44,45 44,51 50,51 50,45" />
          <rect x="43.425" y="31" width="3.6" height="6" stroke="#000000"
            stroke-linecap="round" stroke-linejoin="round" stroke-width="2" />
          <path fill="none" stroke="#000000" stroke-linecap="round" stroke-linejoin="round"
            stroke-width="2" d="M44,45L44,45z" />
          <path fill="none" stroke="#000000" stroke-linecap="round" stroke-linejoin="round"
            stroke-width="2" d="M39,20L39,20z" />
          <path fill="none" stroke="#000000" stroke-linecap="round" stroke-linejoin="round"
            stroke-width="2" d="M16,25L16,25z" />
          <path fill="none" stroke="#000000" stroke-linecap="round" stroke-linejoin="round"
            stroke-width="2" d="M49,25L49,25z" />
          <line x1="28" x2="44" y1="45" y2="45" fill="none" stroke="#000000"
            stroke-linecap="round" stroke-linejoin="round" stroke-width="2" />
          <line x1="32" x2="39" y1="20" y2="20" fill="none" stroke="#000000"
            stroke-linecap="round" stroke-linejoin="round" stroke-width="2" />
          <line x1="16" x2="22" y1="25" y2="25" fill="none" stroke="#000000"
            stroke-linecap="round" stroke-linejoin="round" stroke-width="2" />
          <line x1="49" x2="56" y1="25" y2="25" fill="none" stroke="#000000"
            stroke-linecap="round" stroke-linejoin="round" stroke-width="2" />
        </g>
      </svg>
    </NuxtLink>
  </nav>
</template>

<style scoped>
/* Layout */
nav {
  display: flex;
  gap: 0.45rem;
  justify-content: center;
}

a {
  height: 1.75rem;
}

svg {
  height: 1.55rem;
  margin: 0.10rem 0;
  filter: opacity(70%);
}

/* Theming */
a {
  user-select: none;
}

a[ok="false"] {
  filter: grayscale(100%) opacity(50%);
  pointer-events: none;
}

a:hover svg {
  transform: scale(110%);
  filter: drop-shadow(0 2px 1px oklch(85% 0 0deg));
  /* also clears prior opacity() filter */
}
</style>
