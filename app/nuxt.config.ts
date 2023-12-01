// https://nuxt.com/docs/api/configuration/nuxt-config
const prod = import.meta.env.NODE_ENV == "production";
export default defineNuxtConfig({
  appConfig: {
    clientID: prod ? "794725034152689664" : "1058094051586490368",
    discordGuild: prod ? "793599987694436374" : "1058090773582721214",
  },
  css: [
    "assets/normalize.css",
    "assets/main.css",
  ],
  devtools: { enabled: true },
  nitro: {
    preset: "vercel-edge",
  },
  routeRules: {
    "/api/**": {
      proxy: prod ? "https://huntbot.fly.dev/**" : "http://localhost:8080/**",
    },
    "/login": { ssr: false },
  },
});
