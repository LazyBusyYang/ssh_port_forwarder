<template>
  <div class="h-full flex flex-col space-y-6">
    <!-- 页面标题 -->
    <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
      <h1 class="text-xl sm:text-2xl font-bold text-gray-900">审计日志</h1>
    </div>

    <!-- 过滤器 -->
    <div class="bg-white rounded-lg shadow p-4">
      <div class="flex flex-wrap items-center gap-4">
        <!-- Action 过滤 -->
        <div class="flex items-center gap-2">
          <label class="text-sm font-medium text-gray-700">操作类型:</label>
          <select
            v-model="filters.action"
            class="px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            @change="handleFilterChange"
          >
            <option value="">全部</option>
            <option v-for="action in actionTypes" :key="action" :value="action">
              {{ action }}
            </option>
          </select>
        </div>

        <!-- User ID 过滤 -->
        <div class="flex items-center gap-2">
          <label class="text-sm font-medium text-gray-700">用户 ID:</label>
          <input
            v-model="filters.user_id"
            type="number"
            placeholder="输入用户 ID"
            class="px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 w-32"
            @keyup.enter="handleFilterChange"
          />
        </div>

        <!-- 刷新按钮 -->
        <button
          class="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700 flex items-center gap-2"
          @click="fetchAuditLogs"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          刷新
        </button>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="flex justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
    </div>

    <!-- 数据表格 -->
    <div v-else class="bg-white rounded-lg shadow overflow-hidden">
      <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Time</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">User</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Action</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Target Type</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Target ID</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Detail</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-gray-200">
            <tr v-for="log in auditLogs" :key="log.id" class="hover:bg-gray-50">
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                {{ formatDateTime(log.created_at) }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                <div class="flex items-center gap-2">
                  <div class="w-6 h-6 bg-blue-100 rounded-full flex items-center justify-center">
                    <span class="text-xs font-medium text-blue-600">{{ getUserInitial(log.username) }}</span>
                  </div>
                  <span>{{ log.username || `User #${log.user_id}` }}</span>
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm">
                <span
                  :class="[
                    'px-2 py-1 text-xs font-medium rounded-full',
                    getActionClass(log.action)
                  ]"
                >
                  {{ log.action }}
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                {{ log.target_type || '-' }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                {{ log.target_id || '-' }}
              </td>
              <td class="px-6 py-4 text-sm text-gray-500 max-w-md">
                <div class="truncate" :title="log.detail">
                  {{ log.detail || '-' }}
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 分页 -->
      <div class="px-6 py-4 border-t border-gray-200 flex items-center justify-between">
        <div class="text-sm text-gray-500">
          显示第 {{ (pagination.page - 1) * pagination.page_size + 1 }} 到 
          {{ Math.min(pagination.page * pagination.page_size, pagination.total) }} 条，
          共 {{ pagination.total }} 条
        </div>
        <div class="flex items-center gap-2">
          <button
            :disabled="pagination.page <= 1"
            :class="[
              'px-3 py-1 text-sm font-medium rounded-md',
              pagination.page <= 1
                ? 'bg-gray-100 text-gray-400 cursor-not-allowed'
                : 'bg-white text-gray-700 border border-gray-300 hover:bg-gray-50'
            ]"
            @click="goToPage(pagination.page - 1)"
          >
            上一页
          </button>
          <span class="text-sm text-gray-600">
            第 {{ pagination.page }} / {{ totalPages }} 页
          </span>
          <button
            :disabled="pagination.page >= totalPages"
            :class="[
              'px-3 py-1 text-sm font-medium rounded-md',
              pagination.page >= totalPages
                ? 'bg-gray-100 text-gray-400 cursor-not-allowed'
                : 'bg-white text-gray-700 border border-gray-300 hover:bg-gray-50'
            ]"
            @click="goToPage(pagination.page + 1)"
          >
            下一页
          </button>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-if="!loading && auditLogs.length === 0" class="bg-white rounded-lg shadow p-12 text-center">
      <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <h3 class="mt-4 text-lg font-medium text-gray-900">暂无审计日志</h3>
      <p class="mt-1 text-sm text-gray-500">没有找到符合条件的审计记录</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import api from '../api'

interface AuditLog {
  id: number
  user_id: number
  username: string
  action: string
  target_type: string
  target_id: number
  detail: string
  created_at: number
}

interface Pagination {
  page: number
  page_size: number
  total: number
}

const auditLogs = ref<AuditLog[]>([])
const loading = ref(false)
const filters = ref({
  action: '',
  user_id: ''
})

const pagination = ref<Pagination>({
  page: 1,
  page_size: 20,
  total: 0
})

const actionTypes = [
  'CREATE_HOST',
  'UPDATE_HOST',
  'DELETE_HOST',
  'CREATE_GROUP',
  'UPDATE_GROUP',
  'DELETE_GROUP',
  'CREATE_RULE',
  'UPDATE_RULE',
  'DELETE_RULE',
  'LOGIN',
  'LOGOUT'
]

const totalPages = computed(() => {
  return Math.ceil(pagination.value.total / pagination.value.page_size) || 1
})

onMounted(() => {
  fetchAuditLogs()
})

async function fetchAuditLogs() {
  loading.value = true
  try {
    const params = new URLSearchParams({
      page: String(pagination.value.page),
      page_size: String(pagination.value.page_size)
    })
    
    if (filters.value.action) {
      params.append('action', filters.value.action)
    }
    if (filters.value.user_id) {
      params.append('user_id', filters.value.user_id)
    }
    
    const res = await api.get(`/audit-logs?${params.toString()}`)
    const data = res.data
    
    auditLogs.value = data.data || []
    pagination.value.total = data.total || 0
    pagination.value.page = data.page || 1
    pagination.value.page_size = data.page_size || 20
  } catch (error) {
    console.error('Failed to fetch audit logs:', error)
    auditLogs.value = []
  } finally {
    loading.value = false
  }
}

function handleFilterChange() {
  pagination.value.page = 1
  fetchAuditLogs()
}

function goToPage(page: number) {
  if (page < 1 || page > totalPages.value) return
  pagination.value.page = page
  fetchAuditLogs()
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

function getUserInitial(username: string): string {
  if (!username) return '?'
  return username.charAt(0).toUpperCase()
}

function getActionClass(action: string): string {
  const actionClassMap: Record<string, string> = {
    'CREATE_HOST': 'bg-green-100 text-green-800',
    'UPDATE_HOST': 'bg-blue-100 text-blue-800',
    'DELETE_HOST': 'bg-red-100 text-red-800',
    'CREATE_GROUP': 'bg-green-100 text-green-800',
    'UPDATE_GROUP': 'bg-blue-100 text-blue-800',
    'DELETE_GROUP': 'bg-red-100 text-red-800',
    'CREATE_RULE': 'bg-green-100 text-green-800',
    'UPDATE_RULE': 'bg-blue-100 text-blue-800',
    'DELETE_RULE': 'bg-red-100 text-red-800',
    'LOGIN': 'bg-purple-100 text-purple-800',
    'LOGOUT': 'bg-gray-100 text-gray-800'
  }
  return actionClassMap[action] || 'bg-gray-100 text-gray-800'
}
</script>
