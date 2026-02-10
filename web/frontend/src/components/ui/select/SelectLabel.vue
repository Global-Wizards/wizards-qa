<script setup>
import { SelectLabel } from 'reka-ui'
import { cn } from '@/lib/utils'

defineProps({
  class: { type: String, default: '' },
})
</script>

<template>
  <SelectLabel
    :class="cn('py-1.5 pl-8 pr-2 text-xs font-semibold text-muted-foreground', props.class)"
  >
    <slot />
  </SelectLabel>
</template>
