<!--
  Copyright (c) 2020 Tobias Ahlin. MIT Licensed.
  https://github.com/tobiasahlin/SpinKit (modified)
 -->
<template>
  <div class="spinner" aria-label="Loading" role="status">
    <div class="double-bounce1"></div>
    <div class="double-bounce2"></div>
  </div>
</template>

<style scoped>
.spinner {
  width: 1.25em;
  height: 1.25em;
  position: relative;
  animation: sk-fade 500ms forwards ease-in;
}

.double-bounce1,
.double-bounce2 {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  background-color: oklch(33% 0 0deg);
  opacity: 60%;
  position: absolute;
  top: 0;
  left: 0;
  animation: sk-bounce 2.0s infinite ease-in-out;
}

.double-bounce2 {
  animation-delay: -1.0s;
}

@keyframes sk-bounce {

  0%,
  100% {
    transform: scale(0%);
  }

  50% {
    transform: scale(100%);
  }
}

@keyframes sk-fade {
  0% {
    opacity: 0%;
  }

  100% {
    opacity: 100%;
  }
}
</style>
