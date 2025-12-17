<template>
  <div class="pie-chart-container">
    <div class="bg-white rounded-lg shadow p-6">
      <h3 class="text-lg font-semibold mb-4">{{ title }}</h3>
      <div class="flex items-center justify-center">
        <svg :width="size" :height="size" class="relative">
          <!-- Pie slices -->
          <g v-for="(slice, i) in slices" :key="'slice-' + i">
            <path
              :d="slice.path"
              :fill="slice.color"
              class="cursor-pointer hover:opacity-80 transition-opacity"
              @mouseenter="hoveredIndex = i"
              @mouseleave="hoveredIndex = -1"
            />
            <!-- Label -->
            <text
              v-if="slice.percentage > 5"
              :x="slice.labelX"
              :y="slice.labelY"
              text-anchor="middle"
              class="text-sm font-semibold fill-white"
            >
              {{ slice.percentage.toFixed(0) }}%
            </text>
          </g>
        </svg>
      </div>
      <!-- Legend -->
      <div class="mt-6 grid grid-cols-2 md:grid-cols-3 gap-3">
        <div
          v-for="(item, i) in data"
          :key="i"
          class="flex items-center gap-2 p-2 rounded"
          :class="{ 'bg-gray-100': hoveredIndex === i }"
          @mouseenter="hoveredIndex = i"
          @mouseleave="hoveredIndex = -1"
        >
          <div
            class="w-4 h-4 rounded"
            :style="{ backgroundColor: item.color || colors[i % colors.length] }"
          ></div>
          <span class="text-sm text-gray-700">{{ item.label }}</span>
          <span class="text-sm font-semibold text-gray-900 ml-auto">
            {{ item.value }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

interface PieData {
  label: string
  value: number
  color?: string
}

interface Props {
  data: PieData[]
  title?: string
  size?: number
}

const props = withDefaults(defineProps<Props>(), {
  title: 'Pie Chart',
  size: 300,
})

const hoveredIndex = ref(-1)

const colors = [
  '#3b82f6', // blue
  '#10b981', // green
  '#f59e0b', // yellow
  '#ef4444', // red
  '#8b5cf6', // purple
  '#ec4899', // pink
  '#06b6d4', // cyan
  '#f97316', // orange
]

const total = computed(() => {
  return props.data.reduce((sum, d) => sum + d.value, 0)
})

const slices = computed(() => {
  const centerX = props.size / 2
  const centerY = props.size / 2
  const radius = props.size / 2 - 20

  let currentAngle = -Math.PI / 2 // Start from top

  return props.data.map((item, i) => {
    const percentage = (item.value / total.value) * 100
    const angle = (item.value / total.value) * 2 * Math.PI

    const startAngle = currentAngle
    const endAngle = currentAngle + angle

    const x1 = centerX + radius * Math.cos(startAngle)
    const y1 = centerY + radius * Math.sin(startAngle)
    const x2 = centerX + radius * Math.cos(endAngle)
    const y2 = centerY + radius * Math.sin(endAngle)

    const largeArcFlag = angle > Math.PI ? 1 : 0

    const path = `
      M ${centerX} ${centerY}
      L ${x1} ${y1}
      A ${radius} ${radius} 0 ${largeArcFlag} 1 ${x2} ${y2}
      Z
    `

    // Label position (middle of arc)
    const labelAngle = startAngle + angle / 2
    const labelRadius = radius * 0.7
    const labelX = centerX + labelRadius * Math.cos(labelAngle)
    const labelY = centerY + labelRadius * Math.sin(labelAngle)

    currentAngle = endAngle

    return {
      path,
      color: item.color || colors[i % colors.length],
      percentage,
      labelX,
      labelY,
    }
  })
})
</script>

<style scoped>
.pie-chart-container {
  width: 100%;
}
</style>
