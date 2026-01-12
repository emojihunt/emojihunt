<template>
  <div class="spinner" aria-label="Loading" role="status">
    ⏳
  </div>
</template>

<style scoped>
.spinner {
  width: 1.25em;
  height: 1.25em;
  flex-shrink: 0;

  position: relative;

  animation: spinner 3s ease-in-out infinite,
    spinner-fade 500ms forwards ease-in;
}

@keyframes spinner {
  0% {
    transform: rotate(0deg);
  }

  30%,
  50% {
    transform: rotate(-180deg);
  }

  80%,
  100% {
    transform: rotate(-360deg);
  }
}

@keyframes spinner-fade {
  0% {
    opacity: 0%;
  }

  100% {
    opacity: 80%;
  }
}
</style>
