<script setup>
import { inject, computed } from 'vue'
import { cn } from '@/lib/utils'

defineProps({
  class: { type: String, default: '' },
})

const field = inject('form-field', null)
const message = computed(() => field?.errorMessage?.value)
</script>

<template>
  <p
    v-if="message"
    :class="cn('text-sm font-medium text-destructive', $props.class)"
  >
    {{ message }}
  </p>
</template>
