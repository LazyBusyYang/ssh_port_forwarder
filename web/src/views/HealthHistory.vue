<template>
  <div class="h-full flex flex-col space-y-6">
    <!-- 页面标题 -->
    <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
      <h1 class="text-xl sm:text-2xl font-bold text-gray-900">健康度历史</h1>
    </div>

    <!-- 控制面板 -->
    <div class="bg-white rounded-lg shadow p-4 space-y-4">
      <div class="flex flex-wrap items-center gap-4">
        <!-- Host 选择器 -->
        <div class="flex items-center gap-2">
          <label class="text-sm font-medium text-gray-700">SSH Host:</label>
          <select
            v-model="selectedHostId"
            class="px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 min-w-[200px]"
            @change="fetchHealthHistory"
          >
            <option value="">请选择 Host</option>
            <option v-for="host in hosts" :key="host.id" :value="host.id">
              {{ host.name }} ({{ host.host }}:{{ host.port }})
            </option>
          </select>
        </div>

        <!-- 时间范围快速选择 -->
        <div class="flex items-center gap-2">
          <label class="text-sm font-medium text-gray-700">时间范围:</label>
          <div class="flex rounded-md shadow-sm">
            <button
              v-for="range in timeRanges"
              :key="range.value"
              :class="[
                'px-3 py-2 text-sm font-medium border',
                selectedTimeRange === range.value
                  ? 'bg-blue-600 text-white border-blue-600'
                  : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'
              ]"
              @click="setTimeRange(range.value)"
            >
              {{ range.label }}
            </button>
          </div>
        </div>

        <!-- 自定义时间 -->
        <div v-if="selectedTimeRange === 'custom'" class="flex items-center gap-2">
          <input
            v-model="customStart"
            type="datetime-local"
            class="px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <span class="text-gray-500">至</span>
          <input
            v-model="customEnd"
            type="datetime-local"
            class="px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <button
            class="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700"
            @click="fetchHealthHistory"
          >
            查询
          </button>
        </div>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
    </div>

    <!-- 图表区域 -->
    <div v-else-if="healthData.length > 0" class="space-y-6">
      <!-- 健康度分数图表 -->
      <div class="bg-white rounded-lg shadow p-6">
        <h2 class="text-lg font-semibold text-gray-900 mb-4">健康度分数趋势</h2>
        <div class="h-80">
          <SimpleChart
            :data="scoreData"
            line-color="#3b82f6"
            :threshold="60"
          />
        </div>
        <div class="mt-4 flex items-center gap-4 text-sm">
          <div class="flex items-center gap-2">
            <div class="w-4 h-0.5 bg-blue-500"></div>
            <span class="text-gray-600">健康度分数</span>
          </div>
          <div class="flex items-center gap-2">
            <div class="w-4 h-0.5 bg-red-500 border-dashed" style="border-top: 2px dashed #ef4444;"></div>
            <span class="text-gray-600">阈值 (60分)</span>
          </div>
        </div>
      </div>

      <!-- 延迟图表 -->
      <div class="bg-white rounded-lg shadow p-6">
        <h2 class="text-lg font-semibold text-gray-900 mb-4">检测延迟趋势 (ms)</h2>
        <div class="h-80">
          <SimpleChart
            :data="latencyData"
            line-color="#10b981"
          />
        </div>
        <div class="mt-4 flex items-center gap-4 text-sm">
          <div class="flex items-center gap-2">
            <div class="w-4 h-0.5 bg-emerald-500"></div>
            <span class="text-gray-600">延迟 (ms)</span>
          </div>
        </div>
      </div>

      <!-- 数据表格 -->
      <div class="bg-white rounded-lg shadow overflow-hidden">
        <div class="px-6 py-4 border-b border-gray-200">
          <h2 class="text-lg font-semibold text-gray-900">详细数据</h2>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200">
            <thead class="bg-gray-50">
              <tr>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Time</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Score</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Healthy</th>
                <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Latency (ms)</th>
              </tr>
            </thead>
            <tbody class="bg-white divide-y divide-gray-200">
              <tr v-for="item in healthData" :key="item.id" class="hover:bg-gray-50">
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                  {{ formatDateTime(item.checked_at) }}
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm">
                  <span
                    :class="[
                      'font-medium',
                      item.score >= 60 ? 'text-green-600' : 'text-red-600'
                    ]"
                  >
                    {{ item.score.toFixed(1) }}
                  </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm">
                  <span
                    :class="[
                      'px-2 py-1 text-xs font-medium rounded-full',
                      item.is_healthy
                        ? 'bg-green-100 text-green-800'
                        : 'bg-red-100 text-red-800'
                    ]"
                  >
                    {{ item.is_healthy ? 'Healthy' : 'Unhealthy' }}
                  </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                  {{ item.latency_ms }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-else-if="selectedHostId" class="bg-white rounded-lg shadow p-12 text-center">
      <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
      </svg>
      <h3 class="mt-4 text-lg font-medium text-gray-900">暂无数据</h3>
      <p class="mt-1 text-sm text-gray-500">该时间段内没有健康度检测记录</p>
    </div>

    <!-- 未选择 Host -->
    <div v-else class="bg-white rounded-lg shadow p-12 text-center">
      <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
      </svg>
      <h3 class="mt-4 text-lg font-medium text-gray-900">请选择 SSH Host</h3>
      <p class="mt-1 text-sm text-gray-500">从上方下拉菜单选择一个 Host 查看健康度历史</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import api from '../api'
import SimpleChart from '../components/SimpleChart.vue'

interface SSHHost {
  id: number
  name: string
  host: string
  port: number
}

interface HealthHistoryItem {
  id: number
  host_id: number
  score: number
  is_healthy: boolean
  latency_ms: number
  checked_at: number
}

const hosts = ref<SSHHost[]>([])
const selectedHostId = ref<number | ''>('')
const healthData = ref<HealthHistoryItem[]>([])
const loading = ref(false)

const timeRanges = [
  { label: '1小时', value: '1h' },
  { label: '6小时', value: '6h' },
  { label: '24小时', value: '24h' },
  { label: '7天', value: '7d' },
  { label: '自定义', value: 'custom' },
]

const selectedTimeRange = ref('24h')
const customStart = ref('')
const customEnd = ref('')

const scoreData = computed(() => {
  return healthData.value.map(item => ({
    x: item.checked_at,
    y: item.score
  }))
})

const latencyData = computed(() => {
  return healthData.value.map(item => ({
    x: item.checked_at,
    y: item.latency_ms
  }))
})

onMounted(() => {
  fetchHosts()
  setTimeRange('24h')
})

async function fetchHosts() {
  try {
    const res = await api.get('/hosts?page=1&page_size=100')
    hosts.value = res.data.data || []
  } catch (error) {
    console.error('Failed to fetch hosts:', error)
  }
}

function setTimeRange(range: string) {
  selectedTimeRange.value = range
  
  if (range !== 'custom') {
    const now = Math.floor(Date.now() / 1000)
    let start: number
    
    switch (range) {
      case '1h':
        start = now - 3600
        break
      case '6h':
        start = now - 3600 * 6
        break
      case '24h':
        start = now - 3600 * 24
        break
      case '7d':
        start = now - 3600 * 24 * 7
        break
      default:
        start = now - 3600 * 24
    }
    
    customStart.value = formatDateTimeLocal(start)
    customEnd.value = formatDateTimeLocal(now)
    
    if (selectedHostId.value) {
      fetchHealthHistory()
    }
  }
}

async function fetchHealthHistory() {
  if (!selectedHostId.value) return
  
  loading.value = true
  try {
    const start = Math.floor(new Date(customStart.value).getTime() / 1000)
    const end = Math.floor(new Date(customEnd.value).getTime() / 1000)
    
    const res = await api.get(`/health-history/${selectedHostId.value}?start=${start}&end=${end}&limit=100`)
    healthData.value = res.data.data || []
  } catch (error) {
    console.error('Failed to fetch health history:', error)
    healthData.value = []
  } finally {
    loading.value = false
  }
}

function formatDateTime(timestamp: number): string {
  const date = new Date(timestamp * 1000)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

function formatDateTimeLocal(timestamp: number): string {
  const date = new Date(timestamp * 1000)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hour = String(date.getHours()).padStart(2, '0')
  const minute = String(date.getMinutes()).padStart(2, '0')
  return `${year}-${month}-${day}T${hour}:${minute}`
}
</script>
