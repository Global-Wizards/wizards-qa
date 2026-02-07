<script setup>
import { CheckCircle, Circle, Loader2 } from 'lucide-vue-next'

defineProps({
  status: {
    type: String,
    default: 'pending', // pending, active, complete
  },
  label: {
    type: String,
    required: true,
  },
  detail: {
    type: String,
    default: '',
  },
})
</script>

<template>
  <div class="flex items-start gap-3 py-2">
    <div class="mt-0.5 shrink-0">
      <CheckCircle v-if="status === 'complete'" class="h-5 w-5 text-green-500" />
      <Loader2 v-else-if="status === 'active'" class="h-5 w-5 text-primary animate-spin" />
      <Circle v-else class="h-5 w-5 text-muted-foreground/40" />
    </div>
    <div class="min-w-0">
      <p
        class="text-sm font-medium"
        :class="{
          'text-foreground': status === 'active' || status === 'complete',
          'text-muted-foreground': status === 'pending',
        }"
      >
        {{ label }}
      </p>
      <p v-if="detail" class="text-xs text-muted-foreground mt-0.5 truncate">
        {{ detail }}
      </p>
    </div>
  </div>
</template>
