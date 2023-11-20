// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  devtools: { enabled: true },
  ssr: false,
  runtimeConfig: {
    public: {
      apiBase: 'http://localhost:8080',
      clientID: '1058094051586490368',
    }
  }
})
