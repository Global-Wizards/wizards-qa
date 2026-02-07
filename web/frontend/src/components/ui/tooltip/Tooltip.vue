<script setup>
import { TooltipRoot, TooltipProvider } from 'reka-ui'
</script>

<template>
  <TooltipProvider>
    <TooltipRoot>
      <slot />
    </TooltipRoot>
  </TooltipProvider>
</template>
