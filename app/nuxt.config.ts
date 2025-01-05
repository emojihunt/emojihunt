// https://nuxt.com/docs/api/configuration/nuxt-config
const prod = import.meta.env.NODE_ENV === "production";
export default defineNuxtConfig({
  appConfig: {
    clientID: prod ? "794725034152689664" : "1058094051586490368",
  },
  build: {
    transpile: ["emoji-mart-vue-fast"],
  },
  colorMode: {
    preference: "light",
  },
  compatibilityDate: "2024-12-29",
  css: [
    "assets/normalize.css",
    "assets/main.css",
    "assets/emojimart.css",
  ],
  icon: {
    clientBundle: {
      scan: {
        globInclude: [
          "components/**/*.vue",
          "pages/**/*.vue",
          "node_modules/@nuxt/ui/**/*.js",
        ],
        globExclude: [],
      },
    },
    serverBundle: false,
  },
  nitro: {
    preset: "vercel-edge",
  },
  modules: [
    "@nuxt/icon",
    "@nuxt/ui",
    "@pinia/nuxt",
  ],
  routeRules: {
    "/**": {
      headers: {
        "X-Content-Type-Options": "nosniff",
        "Referrer-Policy": "origin",
        "Cross-Origin-Opener-Policy": "same-origin",
        "Cross-Origin-Embedder-Policy": "require-corp",
        "Cross-Origin-Resource-Policy": "same-site",
      },
    },
    "/api/**": {
      proxy: prod ? "https://huntbot.fly.dev/**" : "http://localhost:8080/**",
    },
  },
  vite: {
    build: {
      assetsInlineLimit: 8192,
    },
  },
});
