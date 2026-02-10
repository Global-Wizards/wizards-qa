<script setup>
import { SelectGroup } from 'reka-ui'

defineProps({
  class: { type: String, default: '' },
})
</script>

<template>
  <SelectGroup :class="class">
    <slot />
  </SelectGroup>
</template>
