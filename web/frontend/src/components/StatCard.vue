<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

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

const displayValue = ref(0)
let rafId = null

function animateValue(target) {
  if (rafId != null) {
    cancelAnimationFrame(rafId)
    rafId = null
  }

  const num = typeof target === 'number' ? target : parseFloat(target)
  if (isNaN(num)) {
    displayValue.value = target
    return
  }

  const duration = 600
  const start = performance.now()
  const startVal = typeof displayValue.value === 'number' ? displayValue.value : 0

  function step(timestamp) {
    const progress = Math.min((timestamp - start) / duration, 1)
    const eased = 1 - Math.pow(1 - progress, 3)
    displayValue.value = Math.round(startVal + (num - startVal) * eased)
    if (progress < 1) {
      rafId = requestAnimationFrame(step)
    } else {
      rafId = null
    }
  }
  rafId = requestAnimationFrame(step)
}

onMounted(() => animateValue(props.value))
onUnmounted(() => {
  if (rafId != null) {
    cancelAnimationFrame(rafId)
    rafId = null
  }
})
watch(() => props.value, animateValue)
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
        <div class="text-2xl font-bold">{{ displayValue }}{{ suffix }}</div>
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
