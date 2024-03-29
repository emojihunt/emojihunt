// https://nuxt.com/docs/api/configuration/nuxt-config
const prod = import.meta.env.NODE_ENV === "production";
export default defineNuxtConfig({
  appConfig: {
    clientID: prod ? "794725034152689664" : "1058094051586490368",
    discordGuild: prod ? "793599987694436374" : "1058090773582721214",
    huntURL: "https://mitmh2024.com/",
  },
  build: {
    transpile: ["emoji-mart-vue-fast"],
  },
  colorMode: {
    preference: "light",
  },
  css: [
    "assets/normalize.css",
    "assets/main.css",
    "assets/emojimart.css",
  ],
  nitro: {
    preset: "vercel-edge",
  },
  modules: [
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
    "/login": { ssr: false },
  },
});
