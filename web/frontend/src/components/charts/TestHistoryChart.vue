<script setup>
import { computed, ref } from 'vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { useTheme } from '@/composables/useTheme'

use([CanvasRenderer, LineChart, GridComponent, TooltipComponent, LegendComponent])

const { isDark } = useTheme()

const props = defineProps({
  data: { type: Array, default: () => [] },
})

const option = computed(() => {
  const dates = props.data.map((d) => d.date)
  const passed = props.data.map((d) => d.passed)
  const failed = props.data.map((d) => d.failed)
  const textColor = isDark.value ? '#94a3b8' : '#64748b'
  const borderColor = isDark.value ? '#1e293b' : '#e2e8f0'

  return {
    tooltip: {
      trigger: 'axis',
      backgroundColor: isDark.value ? '#1e293b' : '#fff',
      borderColor,
      textStyle: { color: textColor },
    },
    legend: {
      data: ['Passed', 'Failed'],
      textStyle: { color: textColor },
    },
    grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
    xAxis: {
      type: 'category',
      data: dates,
      axisLine: { lineStyle: { color: borderColor } },
      axisLabel: { color: textColor },
    },
    yAxis: {
      type: 'value',
      axisLine: { lineStyle: { color: borderColor } },
      axisLabel: { color: textColor },
      splitLine: { lineStyle: { color: borderColor } },
    },
    series: [
      {
        name: 'Passed',
        type: 'line',
        smooth: true,
        data: passed,
        itemStyle: { color: '#10b981' },
        areaStyle: { color: 'rgba(16, 185, 129, 0.1)' },
      },
      {
        name: 'Failed',
        type: 'line',
        smooth: true,
        data: failed,
        itemStyle: { color: '#ef4444' },
        areaStyle: { color: 'rgba(239, 68, 68, 0.1)' },
      },
    ],
  }
})
</script>

<template>
  <VChart :option="option" autoresize style="height: 300px" />
</template>
