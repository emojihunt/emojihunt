<script setup lang="ts">
const config = useAppConfig();
const url = useRequestURL();
const props = defineProps<{
  returnURL?: string;
}>();

const authorize = new URL("https://discord.com/api/oauth2/authorize");
const params = authorize.searchParams;
params.set("client_id", config.clientID);
params.set("redirect_uri", (new URL("/login", url)).toString());
params.set("response_type", "code");
params.set("scope", "identify");
params.set("state", props.returnURL || "/");
</script>

<template>
  <section>
    <h1>🌊🎨🎡</h1>
    <a :href="authorize.toString()">
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
  padding: 1rem;
  margin: 30vh auto;
  max-width: 25rem;
}

h1 {
  text-align: right;
  margin: 0.4rem 0.2rem;
  font-size: 1.2rem;
  letter-spacing: 0.2rem;
  user-select: none;
  opacity: 75%;
  filter: drop-shadow(0 2.5px 4px oklch(77% 0.10 243deg / 40%));
}

a {
  display: flex;
  align-items: center;
  border-radius: 3px;
  height: 3.5rem;

  color: white;
  text-decoration: none;
  user-select: none;
  background-color: oklch(var(--button-oklch));

  box-shadow: 0 0 1px 1px oklch(var(--button-oklch)) inset;
  filter: drop-shadow(0 6px 8px oklch(var(--button-oklch) / 25%));
  background-image: linear-gradient(68deg,
      oklch(100% 0 0deg / 20%) 60%, oklch(0% 0 0deg / 0%) 100%);

  --button-oklch: 58% 0.21 274deg;
}

a:hover {
  --button-oklch: 49% 0.18 274deg;
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
  color: oklch(60% 0.20 24deg)
}
</style>
