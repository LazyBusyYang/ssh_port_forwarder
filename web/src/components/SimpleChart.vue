<template>
  <div class="relative w-full h-full">
    <svg
      :viewBox="`0 0 ${width} ${height}`"
      class="w-full h-full"
      @mouseleave="hoveredIndex = null"
    >
      <!-- 背景网格线 -->
      <g class="grid">
        <!-- 水平网格线 -->
        <line
          v-for="i in 5"
          :key="`h-${i}`"
          :x1="paddingLeft"
          :y1="paddingTop + (chartHeight / 5) * i"
          :x2="width - paddingRight"
          :y2="paddingTop + (chartHeight / 5) * i"
          stroke="#e5e7eb"
          stroke-width="1"
        />
        <!-- 垂直网格线 -->
        <line
          v-for="i in 6"
          :key="`v-${i}`"
          :x1="paddingLeft + (chartWidth / 6) * i"
          :y1="paddingTop"
          :x2="paddingLeft + (chartWidth / 6) * i"
          :y2="height - paddingBottom"
          stroke="#e5e7eb"
          stroke-width="1"
        />
      </g>

      <!-- Y轴标签 -->
      <g class="y-labels">
        <text
          v-for="i in 6"
          :key="`yl-${i}`"
          :x="paddingLeft - 10"
          :y="paddingTop + chartHeight - (chartHeight / 5) * (i - 1) + 4"
          text-anchor="end"
          font-size="11"
          fill="#6b7280"
        >
          {{ Math.round((maxValue / 5) * (i - 1)) }}
        </text>
      </g>

      <!-- X轴标签 -->
      <g class="x-labels">
        <text
          v-for="(label, i) in xLabels"
          :key="`xl-${i}`"
          :x="paddingLeft + (chartWidth / (xLabels.length - 1)) * i"
          :y="height - paddingBottom + 18"
          text-anchor="middle"
          font-size="11"
          fill="#6b7280"
        >
          {{ label }}
        </text>
      </g>

      <!-- 阈值线（如果设置） -->
      <line
        v-if="threshold !== null && threshold !== undefined"
        :x1="paddingLeft"
        :y1="getY(threshold)"
        :x2="width - paddingRight"
        :y2="getY(threshold)"
        stroke="#ef4444"
        stroke-width="2"
        stroke-dasharray="5,5"
      />
      <text
        v-if="threshold !== null && threshold !== undefined"
        :x="width - paddingRight + 5"
        :y="getY(threshold) + 4"
        font-size="10"
        fill="#ef4444"
      >
        {{ threshold }}
      </text>

      <!-- 折线 -->
      <polyline
        :points="points"
        fill="none"
        :stroke="lineColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      />

      <!-- 数据点 -->
      <g class="data-points">
        <circle
          v-for="(point, i) in dataPoints"
          :key="`point-${i}`"
          :cx="point.x"
          :cy="point.y"
          r="4"
          :fill="lineColor"
          :stroke="hoveredIndex === i ? '#fff' : 'none'"
          stroke-width="2"
          class="cursor-pointer transition-all duration-200"
          @mouseenter="hoveredIndex = i"
        />
      </g>

      <!-- 悬停提示 -->
      <g v-if="hoveredIndex !== null && dataPoints[hoveredIndex]">
        <rect
          :x="Math.max(10, Math.min(width - 120, dataPoints[hoveredIndex].x - 60))"
          :y="dataPoints[hoveredIndex].y - 45"
          width="120"
          height="35"
          rx="4"
          fill="#1f2937"
          opacity="0.9"
        />
        <text
          :x="Math.max(10, Math.min(width - 120, dataPoints[hoveredIndex].x - 60)) + 60"
          :y="dataPoints[hoveredIndex].y - 28"
          text-anchor="middle"
          font-size="11"
          fill="#fff"
        >
          {{ formatTime(data[hoveredIndex].x) }}
        </text>
        <text
          :x="Math.max(10, Math.min(width - 120, dataPoints[hoveredIndex].x - 60)) + 60"
          :y="dataPoints[hoveredIndex].y - 15"
          text-anchor="middle"
          font-size="12"
          fill="#fff"
          font-weight="bold"
        >
          {{ data[hoveredIndex].y.toFixed(1) }}
        </text>
      </g>
    </svg>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

interface DataPoint {
  x: number // timestamp
  y: number // value
}

const props = defineProps<{
  data: DataPoint[]
  lineColor?: string
  threshold?: number | null
}>()

const width = 800
const height = 300
const paddingLeft = 50
const paddingRight = 40
const paddingTop = 20
const paddingBottom = 40

const hoveredIndex = ref<number | null>(null)

const chartWidth = computed(() => width - paddingLeft - paddingRight)
const chartHeight = computed(() => height - paddingTop - paddingBottom)

const maxValue = computed(() => {
  if (props.data.length === 0) return 100
  const max = Math.max(...props.data.map(d => d.y))
  return Math.max(max, props.threshold || 0) * 1.1
})

const minValue = computed(() => {
  if (props.data.length === 0) return 0
  return Math.min(...props.data.map(d => d.y)) * 0.9
})

const valueRange = computed(() => maxValue.value - minValue.value)

const dataPoints = computed(() => {
  if (props.data.length === 0) return []
  
  return props.data.map((d, i) => ({
    x: paddingLeft + (i / (props.data.length - 1 || 1)) * chartWidth.value,
    y: paddingTop + chartHeight.value - ((d.y - minValue.value) / valueRange.value) * chartHeight.value,
    original: d
  }))
})

const points = computed(() => {
  return dataPoints.value.map(p => `${p.x},${p.y}`).join(' ')
})

const xLabels = computed(() => {
  if (props.data.length === 0) return []
  const count = Math.min(7, props.data.length)
  const labels: string[] = []
  for (let i = 0; i < count; i++) {
    const index = Math.floor((i / (count - 1)) * (props.data.length - 1))
    labels.push(formatShortTime(props.data[index].x))
  }
  return labels
})

function getY(value: number): number {
  return paddingTop + chartHeight.value - ((value - minValue.value) / valueRange.value) * chartHeight.value
}

function formatTime(timestamp: number): string {
  const date = new Date(timestamp * 1000)
  return date.toLocaleString('zh-CN', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

function formatShortTime(timestamp: number): string {
  const date = new Date(timestamp * 1000)
  const now = new Date()
  const isToday = date.toDateString() === now.toDateString()
  
  if (isToday) {
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  }
  return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}
</script>

<style scoped>
circle:hover {
  r: 6;
}
</style>
