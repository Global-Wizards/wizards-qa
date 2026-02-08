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
  subDetails: {
    type: Array, // Array of { label: string, value: string }
    default: () => [],
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
    <div class="min-w-0 flex-1">
      <p
        class="text-sm font-medium"
        :class="{
          'text-foreground': status === 'active' || status === 'complete',
          'text-muted-foreground': status === 'pending',
        }"
      >
        {{ label }}
      </p>
      <p v-if="detail" class="text-xs text-muted-foreground mt-0.5">
        {{ detail }}
      </p>
      <div v-if="subDetails.length && (status === 'active' || status === 'complete')" class="mt-1.5 space-y-0.5">
        <div
          v-for="(item, i) in subDetails"
          :key="i"
          class="flex items-center gap-1.5 text-xs text-muted-foreground"
        >
          <span class="inline-block w-1 h-1 rounded-full bg-muted-foreground/40 shrink-0" />
          <span v-if="item.label" class="text-muted-foreground/70">{{ item.label }}:</span>
          <span>{{ item.value }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
