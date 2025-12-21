<template>
  <div class="bar-chart-container">
    <div class="bg-white rounded-lg shadow p-6">
      <h3 class="text-lg font-semibold mb-4">{{ title }}</h3>
      <div class="relative" :style="{ height: height + 'px' }">
        <svg :width="width" :height="height" class="w-full">
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

          <!-- Bars -->
          <g v-for="(bar, i) in bars" :key="'bar-' + i">
            <rect
              :x="bar.x"
              :y="bar.y"
              :width="bar.width"
              :height="bar.height"
              :fill="bar.color"
              class="cursor-pointer hover:opacity-80 transition-opacity"
              @mouseenter="hoveredIndex = i"
              @mouseleave="hoveredIndex = -1"
            />
            <!-- Value label on hover -->
            <text
              v-if="hoveredIndex === i"
              :x="bar.x + bar.width / 2"
              :y="bar.y - 5"
              text-anchor="middle"
              class="text-xs font-semibold"
              :fill="bar.color"
            >
              {{ formatValue(bar.value) }}
            </text>
          </g>

          <!-- X-axis labels -->
          <g v-for="(label, i) in xLabels" :key="'label-' + i">
            <text
              :x="label.x"
              :y="height - padding + 20"
              text-anchor="middle"
              class="text-xs text-gray-500"
              :transform="`rotate(-45, ${label.x}, ${height - padding + 20})`"
            >
              {{ label.text }}
            </text>
          </g>
        </svg>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

interface BarData {
  label: string
  value: number
  color?: string
}

interface Props {
  data: BarData[]
  title?: string
  height?: number
  width?: number
  barColor?: string
}

const props = withDefaults(defineProps<Props>(), {
  title: 'Bar Chart',
  height: 300,
  width: 800,
  barColor: '#3b82f6',
})

const hoveredIndex = ref(-1)
const padding = 40
const chartWidth = computed(() => props.width - 2 * padding)
const chartHeight = computed(() => props.height - 2 * padding - 40) // Extra space for rotated labels

const maxValue = computed(() => {
  if (props.data.length === 0) return 100
  const max = Math.max(...props.data.map(d => d.value))
  return Math.ceil(max * 1.1)
})

const yTicks = computed(() => {
  const ticks = []
  const step = maxValue.value / 5
  for (let i = 0; i <= 5; i++) {
    ticks.push({ value: i * step })
  }
  return ticks
})

const barWidth = computed(() => {
  if (props.data.length === 0) return 0
  return (chartWidth.value / props.data.length) * 0.8
})

const barSpacing = computed(() => {
  if (props.data.length === 0) return 0
  return chartWidth.value / props.data.length
})

const bars = computed(() => {
  return props.data.map((d, i) => ({
    x: padding + i * barSpacing.value + (barSpacing.value - barWidth.value) / 2,
    y: padding + chartHeight.value * (1 - d.value / maxValue.value),
    width: barWidth.value,
    height: chartHeight.value * (d.value / maxValue.value),
    value: d.value,
    color: d.color || props.barColor,
  }))
})

const xLabels = computed(() => {
  return props.data.map((d, i) => ({
    x: padding + i * barSpacing.value + barSpacing.value / 2,
    text: d.label.length > 15 ? d.label.substring(0, 15) + '...' : d.label,
  }))
})

const formatValue = (value: number): string => {
  if (value >= 1000) {
    return (value / 1000).toFixed(1) + 'k'
  }
  return value.toFixed(0)
}
</script>

<style scoped>
.bar-chart-container {
  width: 100%;
}
</style>
