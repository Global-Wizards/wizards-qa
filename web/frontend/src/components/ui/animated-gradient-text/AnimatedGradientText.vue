<script setup>
import { cn } from '@/lib/utils'

defineProps({
  class: { type: String, default: '' },
})
</script>

<template>
  <span
    :class="cn(
      'inline-flex animate-gradient bg-gradient-to-r from-[#ffaa40] via-[#9c40ff] to-[#ffaa40] bg-[length:var(--bg-size)_100%] bg-clip-text text-transparent',
      $props.class,
    )"
    style="--bg-size: 300%"
  >
    <slot />
  </span>
</template>

<style scoped>
@keyframes gradient {
  0% { background-position: 0% 50%; }
  50% { background-position: 100% 50%; }
  100% { background-position: 0% 50%; }
}
.animate-gradient {
  animation: gradient 4s ease infinite;
}
</style>
