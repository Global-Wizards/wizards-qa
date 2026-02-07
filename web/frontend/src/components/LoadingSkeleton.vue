<script setup>
import { Skeleton } from '@/components/ui/skeleton'
import { Card, CardHeader, CardContent } from '@/components/ui/card'

const props = defineProps({
  variant: { type: String, default: 'card' },
  count: { type: Number, default: 1 },
})
</script>

<template>
  <!-- Card Skeleton -->
  <template v-if="variant === 'card'">
    <Card v-for="i in count" :key="i">
      <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
        <Skeleton class="h-4 w-24" />
        <Skeleton class="h-4 w-4 rounded" />
      </CardHeader>
      <CardContent>
        <Skeleton class="h-8 w-20 mb-1" />
        <Skeleton class="h-3 w-32" />
      </CardContent>
    </Card>
  </template>

  <!-- Table Row Skeleton -->
  <template v-if="variant === 'table-row'">
    <tr v-for="i in count" :key="i" class="border-b">
      <td class="p-4" v-for="j in 5" :key="j">
        <Skeleton class="h-4 w-full" />
      </td>
    </tr>
  </template>

  <!-- Chart Skeleton -->
  <template v-if="variant === 'chart'">
    <Card v-for="i in count" :key="i">
      <CardHeader>
        <Skeleton class="h-5 w-40" />
      </CardHeader>
      <CardContent>
        <Skeleton class="h-64 w-full rounded" />
      </CardContent>
    </Card>
  </template>
</template>
