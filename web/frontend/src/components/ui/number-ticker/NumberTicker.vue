<script setup>
import { ref, watch, onMounted } from 'vue'
import { useMotion } from '@vueuse/motion'

const props = defineProps({
  value: { type: Number, default: 0 },
  duration: { type: Number, default: 0.6 },
  class: { type: String, default: '' },
})

const displayRef = ref(null)
const displayValue = ref(0)

function animateTo(target) {
  if (!displayRef.value) return
  const from = displayValue.value
  const num = typeof target === 'number' ? target : parseFloat(target)
  if (isNaN(num)) {
    displayValue.value = target
    return
  }

  useMotion(displayRef, {
    initial: { val: from },
    enter: {
      val: num,
      transition: {
        duration: props.duration * 1000,
        ease: 'easeOut',
      },
    },
  })

  // Manually tween using rAF since @vueuse/motion tweens CSS not JS values
  const duration = props.duration * 1000
  const start = performance.now()
  const startVal = from

  function step(timestamp) {
    const progress = Math.min((timestamp - start) / duration, 1)
    const eased = 1 - Math.pow(1 - progress, 3)
    displayValue.value = Math.round(startVal + (num - startVal) * eased)
    if (progress < 1) {
      requestAnimationFrame(step)
    }
  }
  requestAnimationFrame(step)
}

onMounted(() => animateTo(props.value))
watch(() => props.value, animateTo)
</script>

<template>
  <span ref="displayRef" :class="props.class">{{ displayValue }}</span>
</template>
