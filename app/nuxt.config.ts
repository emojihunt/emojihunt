const prod = process.env.NODE_ENV == "production";

// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  appConfig: {
    clientID: prod ? '1058094051586490368' : '1058094051586490368',
  },
  css: [
    "assets/normalize.css",
    "assets/main.css",
  ],
  devtools: { enabled: true },
  nitro: {
    devProxy: {
      "/api": "http://localhost:8080"
    },
  },
});
