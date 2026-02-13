<script setup>
import { cn } from '@/lib/utils'

defineProps({
  class: { type: String, default: '' },
})
</script>

<template>
  <div :class="cn('space-y-2', $props.class)">
    <slot />
  </div>
</template>
