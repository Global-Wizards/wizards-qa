<script setup>
import { computed } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { PieChart } from 'echarts/charts'
import { TooltipComponent, LegendComponent } from 'echarts/components'
import { useTheme } from '@/composables/useTheme'

use([CanvasRenderer, PieChart, TooltipComponent, LegendComponent])

const { isDark } = useTheme()

const props = defineProps({
  passed: { type: Number, default: 0 },
  failed: { type: Number, default: 0 },
})

const option = computed(() => {
  const textColor = isDark.value ? '#94a3b8' : '#64748b'

  return {
    tooltip: {
      trigger: 'item',
      backgroundColor: isDark.value ? '#1e293b' : '#fff',
      borderColor: isDark.value ? '#334155' : '#e2e8f0',
      textStyle: { color: textColor },
    },
    legend: {
      orient: 'horizontal',
      bottom: 0,
      textStyle: { color: textColor },
    },
    series: [
      {
        type: 'pie',
        radius: ['50%', '75%'],
        avoidLabelOverlap: false,
        itemStyle: { borderRadius: 8, borderColor: isDark.value ? '#0f172a' : '#fff', borderWidth: 3 },
        label: {
          show: true,
          position: 'center',
          formatter: () => {
            const total = props.passed + props.failed
            const rate = total > 0 ? Math.round((props.passed / total) * 100) : 0
            return `{value|${rate}%}\n{label|Pass Rate}`
          },
          rich: {
            value: { fontSize: 24, fontWeight: 'bold', color: textColor },
            label: { fontSize: 12, color: textColor, padding: [4, 0, 0, 0] },
          },
        },
        data: [
          { value: props.passed, name: 'Passed', itemStyle: { color: '#10b981' } },
          { value: props.failed, name: 'Failed', itemStyle: { color: '#ef4444' } },
        ],
      },
    ],
  }
})
</script>

<template>
  <VChart :option="option" autoresize style="height: 260px" />
</template>
