<script setup>
import { SelectValue } from 'reka-ui'

const props = defineProps({
  placeholder: { type: String, default: '' },
})
</script>

<template>
  <SelectValue :placeholder="placeholder">
    <slot />
  </SelectValue>
</template>
