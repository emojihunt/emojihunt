// https://nuxt.com/docs/api/configuration/nuxt-config
const prod = import.meta.env.NODE_ENV === "production";
export default defineNuxtConfig({
  appConfig: {
    apiBase: prod ? "https://api.emojihunt.org" : "http://localhost:8080",
    clientID: prod ? "794725034152689664" : "1058094051586490368",
    newSyncBackend: true,
  },
  build: {
    transpile: ["emoji-mart-vue-fast", "linkify", "remarkable"],
  },
  colorMode: {
    preference: "light",
  },
  compatibilityDate: "2025-07-15",
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
    preset: "bun",
  },
  modules: [
    "@nuxt/icon",
    "@nuxt/ui",
  ],
  routeRules: {
    "/**": {
      headers: {
        "Cross-Origin-Opener-Policy": "same-origin",
        "Cross-Origin-Resource-Policy": "same-site",
        "Referrer-Policy": "origin",
        "X-Content-Type-Options": "nosniff",
      },
    },
    "/": { prerender: false }, // otherwise, just redirects to /login
  },
  vite: {
    build: {
      assetsInlineLimit: 8192,
      chunkSizeWarningLimit: 1024,
      rollupOptions: {
        output: {
          manualChunks(id) {
            // Emit large JSON files as separate chunks
            if (id.includes("assets/emojimart.json")) {
              return "emojimart-json";
            } else if (id.includes("assets/emoji-metadata.json")) {
              return "emoji-metadata-json";
            }
            return null;
          },
        },
      },
    },
  },
});
