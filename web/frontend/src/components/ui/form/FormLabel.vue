<script setup>
import { inject, computed } from 'vue'
import { cn } from '@/lib/utils'

defineProps({
  class: { type: String, default: '' },
})

const field = inject('form-field', null)
const hasError = computed(() => field?.errorMessage?.value)
</script>

<template>
  <label
    :class="cn(
      'text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70',
      hasError && 'text-destructive',
      $props.class,
    )"
  >
    <slot />
  </label>
</template>
