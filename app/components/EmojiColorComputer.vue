<script setup lang="ts">
import Color from "colorjs.io";
const { emoji, alternates } = defineProps<{ emoji: string; alternates: string[]; }>();
const canvas = useTemplateRef("canvas");
const median = ref(-1);
const result = ref<any[]>([]);
defineExpose({ result });

onMounted(() => {
  const cv = canvas.value!;
  const ctx = cv.getContext("2d")!;
  ctx.clearRect(0, 0, cv.width, cv.height);
  ctx.font = `${cv.width / 2}px "Noto Color Emoji"`;
  ctx.textAlign = "center";
  ctx.textBaseline = "middle";
  ctx.fillText(emoji + "\ufe0f", cv.width / 2, cv.height / 2);

  setTimeout(() => {
    const { colorSpace, data } = ctx.getImageData(0, 0, cv.width, cv.height);
    const hues = [];
    for (let i = 0; i < data.length; i += 4) {
      const color = new Color({
        spaceId: colorSpace,
        coords: [data[i], data[i + 1], data[i + 2]],
        alpha: data[i + 3],
      });
      const [lightness, chroma, hue] = color.oklch;
      if (!(lightness + chroma + hue)) continue;
      const weight = Math.round(color.alpha * chroma * 10);
      for (let i = 0; i < weight; i++) hues.push(hue);
    }
    median.value = Math.round(circularMean(hues));
    result.value = [median.value, emoji, ...alternates];
  }, 0);
});

// From https://stackoverflow.com/a/63843172. CC-BY-SA 4.0.
const period = 360;
function circularMean(inputs: number[]) {
  var scalingFactor = 2 * Math.PI / period;
  var sines = 0.0;
  var cosines = 0.0;
  for (const value of inputs) {
    var radians = value * scalingFactor;
    sines += Math.sin(radians);
    cosines += Math.cos(radians);
  }
  var circularMean = Math.atan2(sines, cosines) / scalingFactor;
  if (circularMean >= 0)
    return circularMean;
  else
    return circularMean + period;
}
</script>

<template>
  <div :class="median >= 0 && 'ok'">
    <canvas ref="canvas" width="20" height="20"></canvas>
    <span>{{ median }}</span>
  </div>
</template>

<style scoped>
div {
  width: 60px;
  display: flex;
  flex-direction: column;
  border: 5px solid lightgray;
}

span {
  padding-bottom: 3px;
  text-align: center;
  font-weight: 750;
  font-feature-settings: "tnum";
  color: oklch(75% 0.2 v-bind(median));
  visibility: hidden;
}

div.ok {
  border-color: oklch(75% 0.2 v-bind(median));
}

div.ok span {
  visibility: unset;
}
</style>
