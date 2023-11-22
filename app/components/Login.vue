<script setup lang="ts">
const props = defineProps<{
  return: string;
}>();

const authenticate = new URL("https://discord.com/api/oauth2/authorize");
const params = authenticate.searchParams;

const config = useAppConfig();
const url = useRequestURL();
const state = { r: props.return };
params.set("client_id", config.clientID);
params.set("redirect_uri", (new URL("/login", url)).toString());
params.set("response_type", "code");
params.set("scope", "identify");
params.set("state", JSON.stringify(state));
</script>

<template>
  <section>
    <h1>🌊🎨🎡</h1>
    <a :href="authenticate.toString()">
      <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" viewBox="0 0 16 16">
        <path
          d="M13.545 2.907a13.227 13.227 0 0 0-3.257-1.011.05.05 0 0 0-.052.025c-.141.25-.297.577-.406.833a12.19 12.19 0 0 0-3.658 0 8.258 8.258 0 0 0-.412-.833.051.051 0 0 0-.052-.025c-1.125.194-2.22.534-3.257 1.011a.041.041 0 0 0-.021.018C.356 6.024-.213 9.047.066 12.032c.001.014.01.028.021.037a13.276 13.276 0 0 0 3.995 2.02.05.05 0 0 0 .056-.019c.308-.42.582-.863.818-1.329a.05.05 0 0 0-.01-.059.051.051 0 0 0-.018-.011 8.875 8.875 0 0 1-1.248-.595.05.05 0 0 1-.02-.066.051.051 0 0 1 .015-.019c.084-.063.168-.129.248-.195a.05.05 0 0 1 .051-.007c2.619 1.196 5.454 1.196 8.041 0a.052.052 0 0 1 .053.007c.08.066.164.132.248.195a.051.051 0 0 1-.004.085 8.254 8.254 0 0 1-1.249.594.05.05 0 0 0-.03.03.052.052 0 0 0 .003.041c.24.465.515.909.817 1.329a.05.05 0 0 0 .056.019 13.235 13.235 0 0 0 4.001-2.02.049.049 0 0 0 .021-.037c.334-3.451-.559-6.449-2.366-9.106a.034.034 0 0 0-.02-.019Zm-8.198 7.307c-.789 0-1.438-.724-1.438-1.612 0-.889.637-1.613 1.438-1.613.807 0 1.45.73 1.438 1.613 0 .888-.637 1.612-1.438 1.612Zm5.316 0c-.788 0-1.438-.724-1.438-1.612 0-.889.637-1.613 1.438-1.613.807 0 1.451.73 1.438 1.613 0 .888-.631 1.612-1.438 1.612Z" />
      </svg>
      <div class="vh"></div>
      <span>Log in</span>
    </a>
    <div class="error">
      <slot />
    </div>
  </section>
</template>

<style scoped>
section {
  margin: 30vh auto;
  max-width: 25rem;
}

h1 {
  text-align: right;
  margin: 0.4rem 0.2rem;
  font-size: 1.2rem;
  letter-spacing: 0.2rem;
  opacity: 75%;
  filter: drop-shadow(0.9px 2.3px 4px hsl(206deg 80% 70% / 40%));
}

a {
  display: flex;
  align-items: center;
  border-radius: 3px;
  height: 3.5rem;

  color: white;
  text-decoration: none;
  background-color: hsl(var(--button-hsl));
  box-shadow: 0 0 1px 1px hsl(235 100% 65% / 70%) inset,
    0.9px 2.3px 2.2px hsl(var(--button-hsl) / 5%),
    2.5px 6.3px 6.1px hsl(var(--button-hsl) / 7%),
    6px 15.1px 14.8px hsl(var(--button-hsl) / 10%);
  --button-hsl: 235 86% 65%;

  background-image: linear-gradient(68deg, hsla(0, 0%, 100%, 20%) 60%, hsla(0, 0%, 100%, 0) 100%);
}

a:hover {
  --button-hsl: 235 51% 52%;
}

a svg {
  width: auto;
  height: 2rem;
  padding: 0.2rem 2rem 0;
}

a .vh {
  height: 2.75rem;
  border-left: 1px solid white;
}

a span {
  font-weight: 600;
  font-size: 1.1rem;
  line-height: 2.5rem;
  padding: 0.2rem 1.5rem 0;
}

.error {
  padding: 1rem 0.2rem;
  color: hsl(0, 74%, 60%);
}
</style>
