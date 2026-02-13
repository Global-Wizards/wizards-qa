<script setup>
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { NumberTicker } from '@/components/ui/number-ticker'

const props = defineProps({
  title: { type: String, required: true },
  value: { type: [Number, String], required: true },
  icon: { type: Object, default: null },
  trend: { type: String, default: null },
  trendUp: { type: Boolean, default: true },
  suffix: { type: String, default: '' },
  description: { type: String, default: null },
  to: { type: String, default: null },
  iconColor: { type: String, default: 'text-muted-foreground' },
})
</script>

<template>
  <component
    :is="to ? 'router-link' : 'div'"
    :to="to || undefined"
    :class="to ? 'block group' : ''"
  >
    <Card :class="[
      'transition-all duration-200',
      to && 'cursor-pointer hover:shadow-md hover:border-primary/20'
    ]">
      <CardHeader class="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle class="text-sm font-medium">{{ title }}</CardTitle>
        <component
          :is="icon"
          v-if="icon"
          class="h-4 w-4 transition-transform duration-200"
          :class="[iconColor, to && 'group-hover:-translate-y-0.5']"
        />
      </CardHeader>
      <CardContent>
        <div class="text-2xl font-bold">
          <NumberTicker :value="typeof value === 'number' ? value : parseFloat(value) || 0" :duration="0.6" />{{ suffix }}
        </div>
        <p v-if="trend" class="text-xs text-muted-foreground mt-1">
          <span :class="trendUp ? 'text-emerald-500' : 'text-red-500'">
            {{ trendUp ? '+' : '' }}{{ trend }}
          </span>
          from last period
        </p>
        <p v-else-if="description" class="text-xs text-muted-foreground mt-1">
          {{ description }}
        </p>
      </CardContent>
    </Card>
  </component>
</template>
