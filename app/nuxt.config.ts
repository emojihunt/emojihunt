// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  appConfig: {
    clientID: import.meta.env.PROD ? '1058094051586490368' : '1058094051586490368',
  },
  css: [
    "assets/normalize.css",
    "assets/main.css",
  ],
  devtools: { enabled: true },
  routeRules: {
    "/api/**": { proxy: "http://localhost:8080/**" },
    "/login": { ssr: false },
  },
});
