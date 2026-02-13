<script setup>
import { cn } from '@/lib/utils'

defineProps({
  class: { type: String, default: '' },
  shimmerColor: { type: String, default: '#ffffff' },
  shimmerSize: { type: String, default: '0.05em' },
  shimmerDuration: { type: String, default: '3s' },
  borderRadius: { type: String, default: '100px' },
  background: { type: String, default: 'hsl(var(--primary))' },
})
</script>

<template>
  <button
    :class="cn(
      'shimmer-button group relative z-0 inline-flex cursor-pointer items-center justify-center overflow-hidden whitespace-nowrap px-6 py-3 text-sm font-medium text-white transition-all [border-radius:var(--radius)] hover:scale-105 active:scale-95',
      $props.class,
    )"
    :style="{
      '--shimmer-color': shimmerColor,
      '--shimmer-size': shimmerSize,
      '--shimmer-duration': shimmerDuration,
      '--radius': borderRadius,
      '--bg': background,
    }"
  >
    <!-- shimmer -->
    <span class="shimmer-slide absolute inset-0 overflow-hidden [border-radius:var(--radius)]">
      <span
        class="shimmer-effect absolute inset-[-100%] animate-shimmer-slide"
        :style="{
          background: `conic-gradient(from 0deg, transparent 0 340deg, var(--shimmer-color) 360deg)`,
        }"
      />
    </span>
    <!-- backdrop -->
    <span class="shimmer-backdrop absolute inset-[1px] [border-radius:var(--radius)]" :style="{ background: 'var(--bg)' }" />
    <!-- content -->
    <span class="z-10 flex items-center gap-2">
      <slot />
    </span>
  </button>
</template>

<style scoped>
@keyframes shimmer-slide {
  to {
    transform: translate(calc(100cqw - 100%), 0);
  }
}

.shimmer-slide {
  container-type: inline-size;
}

.animate-shimmer-slide {
  animation: shimmer-slide var(--shimmer-duration) ease-in-out infinite alternate;
}

.shimmer-button {
  box-shadow: 0 0 12px -2px var(--bg);
  transition: box-shadow 0.3s, transform 0.2s;
}

.shimmer-button:hover {
  box-shadow: 0 0 20px -2px var(--bg);
}
</style>
