<template>
  <div class="multi-line-chart-container">
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

          <!-- Lines for each dataset -->
          <g v-for="(dataset, datasetIdx) in datasets" :key="datasetIdx">
            <path
              :d="getLinePath(dataset)"
              :stroke="dataset.color"
              stroke-width="2"
              fill="none"
              class="transition-all duration-300"
            />
            <!-- Data points -->
            <g v-for="(point, i) in getPoints(dataset)" :key="'point-' + datasetIdx + '-' + i">
              <circle
                :cx="point.x"
                :cy="point.y"
                r="4"
                :fill="dataset.color"
                class="cursor-pointer hover:r-6 transition-all"
              />
            </g>
          </g>

          <!-- X-axis labels -->
          <g v-for="(label, i) in xLabels" :key="'label-' + i">
            <text
              :x="label.x"
              :y="height - padding + 20"
              text-anchor="middle"
              class="text-xs text-gray-500"
            >
              {{ label.text }}
            </text>
          </g>
        </svg>
      </div>
      <div v-if="legend" class="mt-4 flex justify-center gap-4 flex-wrap">
        <div v-for="(item, i) in legend" :key="i" class="flex items-center gap-2">
          <div class="w-3 h-3 rounded" :style="{ backgroundColor: item.color }"></div>
          <span class="text-sm text-gray-600">{{ item.label }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface DataPoint {
  date: string
  value: number
}

interface Dataset {
  label: string
  data: DataPoint[]
  color: string
}

interface Props {
  datasets: Dataset[]
  title?: string
  height?: number
  width?: number
}

const props = withDefaults(defineProps<Props>(), {
  title: 'Chart',
  height: 300,
  width: 800,
})

const padding = 40
const chartWidth = computed(() => props.width - 2 * padding)
const chartHeight = computed(() => props.height - 2 * padding)

const maxValue = computed(() => {
  let max = 0
  props.datasets.forEach(dataset => {
    const datasetMax = Math.max(...dataset.data.map(d => d.value))
    max = Math.max(max, datasetMax)
  })
  return Math.ceil(max * 1.1) || 100
})

const yTicks = computed(() => {
  const ticks = []
  const step = maxValue.value / 5
  for (let i = 0; i <= 5; i++) {
    ticks.push({ value: i * step })
  }
  return ticks
})

const allDates = computed(() => {
  const dates = new Set<string>()
  props.datasets.forEach(dataset => {
    dataset.data.forEach(d => dates.add(d.date))
  })
  return Array.from(dates).sort()
})

const getPoints = (dataset: Dataset) => {
  if (dataset.data.length === 0) return []

  const step = chartWidth.value / (allDates.value.length - 1 || 1)

  return dataset.data.map(d => {
    const xIndex = allDates.value.indexOf(d.date)
    return {
      x: padding + xIndex * step,
      y: padding + chartHeight.value * (1 - d.value / maxValue.value),
    }
  })
}

const getLinePath = (dataset: Dataset) => {
  const points = getPoints(dataset)
  if (points.length === 0) return ''

  let path = `M ${points[0].x} ${points[0].y}`
  for (let i = 1; i < points.length; i++) {
    path += ` L ${points[i].x} ${points[i].y}`
  }
  return path
}

const xLabels = computed(() => {
  if (allDates.value.length === 0) return []

  const step = chartWidth.value / (allDates.value.length - 1 || 1)
  const labelStep = Math.max(1, Math.floor(allDates.value.length / 5))

  return allDates.value
    .filter((_, i) => i % labelStep === 0 || i === allDates.value.length - 1)
    .map((date, idx) => ({
      x: padding + idx * labelStep * step,
      text: formatDate(date),
    }))
})

const legend = computed(() => {
  return props.datasets.map(d => ({
    label: d.label,
    color: d.color,
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
.multi-line-chart-container {
  width: 100%;
}
</style>
