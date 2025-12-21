<template>
  <div class="line-chart-container">
    <div class="bg-white rounded-lg shadow p-6">
      <h3 class="text-lg font-semibold mb-4">{{ title }}</h3>
      <div class="relative" :style="{ height: height + 'px' }">
        <svg :width="width" :height="height" class="w-full">
          <!-- Grid lines -->
          <defs>
            <linearGradient id="lineGradient" x1="0%" y1="0%" x2="0%" y2="100%">
              <stop offset="0%" :style="{ stopColor: lineColor, stopOpacity: 0.3 }" />
              <stop offset="100%" :style="{ stopColor: lineColor, stopOpacity: 0 }" />
            </linearGradient>
          </defs>

          <!-- Y-axis grid lines -->
          <g v-for="(tick, i) in yTicks" :key="'grid-' + i">
            <line
              :x1="padding"
              :y1="padding + (height - 2 * padding) * (1 - tick.value / maxValue)"
              :x2="width - padding"
              :y2="padding + (height - 2 * padding) * (1 - tick.value / maxValue)"
              stroke="#e5e7eb"
              stroke-width="1"
            />
            <text
              :x="padding - 10"
              :y="padding + (height - 2 * padding) * (1 - tick.value / maxValue) + 4"
              text-anchor="end"
              class="text-xs text-gray-500"
            >
              {{ formatValue(tick.value) }}
            </text>
          </g>

          <!-- Area fill -->
          <path
            :d="areaPath"
            fill="url(#lineGradient)"
          />

          <!-- Line -->
          <path
            :d="linePath"
            :stroke="lineColor"
            stroke-width="2"
            fill="none"
            class="transition-all duration-300"
          />

          <!-- Data points -->
          <g v-for="(point, i) in points" :key="'point-' + i">
            <circle
              :cx="point.x"
              :cy="point.y"
              r="4"
              :fill="lineColor"
              class="cursor-pointer hover:r-6 transition-all"
            />
            <title>{{ point.label }}</title>
          </g>

          <!-- X-axis labels -->
          <g v-for="(point, i) in xLabels" :key="'label-' + i">
            <text
              :x="point.x"
              :y="height - padding + 20"
              text-anchor="middle"
              class="text-xs text-gray-500"
            >
              {{ point.label }}
            </text>
          </g>
        </svg>
      </div>
      <div v-if="legend" class="mt-4 flex justify-center gap-4">
        <div v-for="(item, i) in legend" :key="i" class="flex items-center gap-2">
          <div class="w-3 h-3 rounded" :style="{ backgroundColor: item.color }"></div>
          <span class="text-sm text-gray-600">{{ item.label }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';

interface Props {
  data: Array<{ date: string; value: number }>
  title?: string
  height?: number
  width?: number
  lineColor?: string
  legend?: Array<{ label: string; color: string }>
}

const props = withDefaults(defineProps<Props>(), {
  title: 'Chart',
  height: 300,
  width: 800,
  lineColor: '#3b82f6',
  legend: undefined,
})

const padding = 40
const chartWidth = computed(() => props.width - 2 * padding)
const chartHeight = computed(() => props.height - 2 * padding)

const maxValue = computed(() => {
  if (props.data.length === 0) return 100
  const max = Math.max(...props.data.map(d => d.value))
  return Math.ceil(max * 1.1) // Add 10% padding
})

const yTicks = computed(() => {
  const ticks = []
  const step = maxValue.value / 5
  for (let i = 0; i <= 5; i++) {
    ticks.push({ value: i * step })
  }
  return ticks
})

const points = computed(() => {
  if (props.data.length === 0) return []

  const step = chartWidth.value / (props.data.length - 1 || 1)

  return props.data.map((d, i) => ({
    x: padding + i * step,
    y: padding + chartHeight.value * (1 - d.value / maxValue.value),
    label: `${d.date}: ${formatValue(d.value)}`,
  }))
})

const linePath = computed(() => {
  if (points.value.length === 0) return ''

  let path = `M ${points.value[0].x} ${points.value[0].y}`
  for (let i = 1; i < points.value.length; i++) {
    path += ` L ${points.value[i].x} ${points.value[i].y}`
  }
  return path
})

const areaPath = computed(() => {
  if (points.value.length === 0) return ''

  const bottomY = padding + chartHeight.value
  let path = `M ${points.value[0].x} ${bottomY} L ${points.value[0].x} ${points.value[0].y}`

  for (let i = 1; i < points.value.length; i++) {
    path += ` L ${points.value[i].x} ${points.value[i].y}`
  }

  path += ` L ${points.value[points.value.length - 1].x} ${bottomY} Z`
  return path
})

const xLabels = computed(() => {
  if (props.data.length === 0) return []

  const step = chartWidth.value / (props.data.length - 1 || 1)
  const labelStep = Math.max(1, Math.floor(props.data.length / 5))

  return props.data
    .filter((_, i) => i % labelStep === 0 || i === props.data.length - 1)
    .map((d, idx) => ({
      x: padding + idx * labelStep * step,
      label: formatDate(d.date),
    }))
})

const formatValue = (value: number): string => {
  if (value >= 1000) {
    return (value / 1000).toFixed(1) + 'k'
  }
  return value.toFixed(0)
}

const formatDate = (date: string): string => {
  const d = new Date(date)
  return `${d.getMonth() + 1}/${d.getDate()}`
}
</script>

<style scoped>
.line-chart-container {
  width: 100%;
}
</style>
