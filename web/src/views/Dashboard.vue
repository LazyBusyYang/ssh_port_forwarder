<template>
  <div class="p-6">
    <h1 class="text-2xl font-bold text-gray-800 mb-6">Dashboard</h1>

    <!-- 统计卡片行 -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4 mb-8">
      <!-- Host 总数 -->
      <div class="bg-white rounded-lg shadow p-5 border-l-4 border-blue-500">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-gray-500 mb-1">Host 总数</p>
            <p class="text-2xl font-bold text-gray-800">{{ overview.total_hosts }}</p>
          </div>
          <div class="w-12 h-12 bg-blue-100 rounded-lg flex items-center justify-center">
            <svg class="w-6 h-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
            </svg>
          </div>
        </div>
      </div>

      <!-- 健康 Host 数 -->
      <div class="bg-white rounded-lg shadow p-5 border-l-4 border-green-500">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-gray-500 mb-1">健康 Host</p>
            <p class="text-2xl font-bold text-green-600">{{ overview.healthy_hosts }}</p>
          </div>
          <div class="w-12 h-12 bg-green-100 rounded-lg flex items-center justify-center">
            <svg class="w-6 h-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
        </div>
      </div>

      <!-- Rule 总数 -->
      <div class="bg-white rounded-lg shadow p-5 border-l-4 border-purple-500">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-gray-500 mb-1">Rule 总数</p>
            <p class="text-2xl font-bold text-gray-800">{{ overview.total_rules }}</p>
          </div>
          <div class="w-12 h-12 bg-purple-100 rounded-lg flex items-center justify-center">
            <svg class="w-6 h-6 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
            </svg>
          </div>
        </div>
      </div>

      <!-- 活跃 Rule 数 -->
      <div class="bg-white rounded-lg shadow p-5 border-l-4 border-orange-500">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-gray-500 mb-1">活跃 Rules</p>
            <p class="text-2xl font-bold text-orange-600">{{ overview.active_rules }}</p>
          </div>
          <div class="w-12 h-12 bg-orange-100 rounded-lg flex items-center justify-center">
            <svg class="w-6 h-6 text-orange-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
            </svg>
          </div>
        </div>
      </div>

      <!-- 活跃连接数 -->
      <div class="bg-white rounded-lg shadow p-5 border-l-4 border-cyan-500">
        <div class="flex items-center justify-between">
          <div>
            <p class="text-sm text-gray-500 mb-1">活跃连接</p>
            <p class="text-2xl font-bold text-cyan-600">{{ overview.active_connections }}</p>
          </div>
          <div class="w-12 h-12 bg-cyan-100 rounded-lg flex items-center justify-center">
            <svg class="w-6 h-6 text-cyan-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 11.5V14m0-2.5v-6a1.5 1.5 0 113 0m-3 6a1.5 1.5 0 00-3 0v2a7.5 7.5 0 0015 0v-5a1.5 1.5 0 00-3 0m-6-3V11m0-5.5v-1a1.5 1.5 0 013 0v1m0 0V11m0-5.5a1.5 1.5 0 013 0v3m0 0V11" />
            </svg>
          </div>
        </div>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="flex justify-center items-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      <span class="ml-3 text-gray-600">加载中...</span>
    </div>

    <!-- 错误提示 -->
    <div v-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700">
      {{ error }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '../api'

interface OverviewData {
  total_hosts: number
  healthy_hosts: number
  total_rules: number
  active_rules: number
  active_connections: number
}

const overview = ref<OverviewData>({
  total_hosts: 0,
  healthy_hosts: 0,
  total_rules: 0,
  active_rules: 0,
  active_connections: 0
})

const loading = ref(false)
const error = ref('')

const fetchOverview = async () => {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get('/status/overview')
    if (res.data.code === 0) {
      overview.value = res.data.data
    } else {
      error.value = res.data.message || '获取数据失败'
    }
  } catch (err: any) {
    error.value = err.response?.data?.message || '网络错误，请稍后重试'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchOverview()
})
</script>
