<script setup>
import { computed } from 'vue'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

const props = defineProps({
  status: { type: String, required: true },
})

const statusConfig = {
  passed: { label: 'Passed', class: 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-400 border-emerald-500/20' },
  failed: { label: 'Failed', class: 'bg-red-500/15 text-red-700 dark:text-red-400 border-red-500/20' },
  timeout: { label: 'Timeout', class: 'bg-yellow-500/15 text-yellow-700 dark:text-yellow-400 border-yellow-500/20' },
  running: { label: 'Running', class: 'bg-blue-500/15 text-blue-700 dark:text-blue-400 border-blue-500/20' },
  pending: { label: 'Pending', class: 'bg-gray-500/15 text-gray-700 dark:text-gray-400 border-gray-500/20' },
}

const config = computed(() => statusConfig[props.status] || statusConfig.pending)
</script>

<template>
  <Badge variant="outline" :class="config.class">
    {{ config.label }}
  </Badge>
</template>
