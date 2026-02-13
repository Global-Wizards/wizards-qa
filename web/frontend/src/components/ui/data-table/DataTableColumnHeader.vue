<script setup>
import { ArrowUp, ArrowDown, ArrowUpDown } from 'lucide-vue-next'
import { cn } from '@/lib/utils'

const props = defineProps({
  column: { type: Object, required: true },
  title: { type: String, required: true },
  class: { type: String, default: '' },
})

function toggleSorting() {
  props.column.toggleSorting()
}
</script>

<template>
  <div
    v-if="column.getCanSort()"
    :class="cn('flex items-center gap-1 cursor-pointer select-none', props.class)"
    @click="toggleSorting"
  >
    {{ title }}
    <ArrowUp v-if="column.getIsSorted() === 'asc'" class="h-3.5 w-3.5" />
    <ArrowDown v-else-if="column.getIsSorted() === 'desc'" class="h-3.5 w-3.5" />
    <ArrowUpDown v-else class="h-3.5 w-3.5 text-muted-foreground/50" />
  </div>
  <div v-else :class="props.class">{{ title }}</div>
</template>
