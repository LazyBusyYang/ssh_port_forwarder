<template>
  <div class="h-full flex flex-col space-y-6">
    <!-- 页面标题 -->
    <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
      <h1 class="text-xl sm:text-2xl font-bold text-gray-900">Forward Rules</h1>
      <button
        @click="openCreateModal"
        class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors flex items-center space-x-2 whitespace-nowrap"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        <span>创建转发规则</span>
      </button>
    </div>

    <!-- 错误提示 -->
    <div v-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4 flex items-center space-x-3">
      <svg class="w-5 h-5 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <span class="text-red-700">{{ error }}</span>
      <button @click="fetchRules" class="text-red-600 hover:text-red-800 underline ml-auto">重试</button>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="flex items-center justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      <span class="ml-3 text-gray-500">加载中...</span>
    </div>

    <!-- 空状态 -->
    <div v-else-if="rules.length === 0" class="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center">
      <svg class="w-16 h-16 text-gray-300 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
      </svg>
      <h3 class="text-lg font-medium text-gray-900 mb-1">暂无转发规则</h3>
      <p class="text-gray-500 mb-4">创建第一个转发规则来配置端口转发</p>
      <button
        @click="openCreateModal"
        class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
      >
        创建转发规则
      </button>
    </div>

    <!-- 表格 - 添加水平滚动 -->
    <div v-else class="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden flex-1 flex flex-col min-h-0">
      <div class="overflow-x-auto flex-1">
        <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Local Port</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Target</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Protocol</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Group</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Active Host</th>
            <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">操作</th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-for="rule in rules" :key="rule.id" class="hover:bg-gray-50">
            <td class="px-6 py-4 whitespace-nowrap font-medium text-gray-900">
              {{ rule.local_port }}
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-gray-700">
              {{ rule.target_host }}:{{ rule.target_port }}
            </td>
            <td class="px-6 py-4 whitespace-nowrap">
              <span class="px-2 py-1 text-xs rounded-full bg-gray-100 text-gray-800 uppercase">
                {{ rule.protocol }}
              </span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-gray-700">
              {{ rule.group?.name || '-' }}
            </td>
            <td class="px-6 py-4 whitespace-nowrap">
              <span class="px-2 py-1 text-xs rounded-full" :class="getStatusClass(rule.status)">
                {{ getStatusLabel(rule.status) }}
              </span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-gray-700">
              {{ rule.active_host?.name || '-' }}
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-right space-x-2">
              <button
                @click="restartRule(rule.id)"
                :disabled="restartingRuleId === rule.id"
                class="text-green-600 hover:text-green-800 text-sm"
              >
                {{ restartingRuleId === rule.id ? '重启中...' : '重启' }}
              </button>
              <button
                @click="openEditModal(rule)"
                class="text-indigo-600 hover:text-indigo-800 text-sm"
              >
                编辑
              </button>
              <button
                @click="confirmDelete(rule)"
                class="text-red-600 hover:text-red-800 text-sm"
              >
                删除
              </button>
            </td>
          </tr>
        </tbody>
        </table>
      </div>

      <!-- 分页 -->
      <div class="bg-gray-50 px-4 sm:px-6 py-3 flex flex-col sm:flex-row items-center justify-between border-t border-gray-200 gap-2">
        <div class="text-sm text-gray-500">
          共 {{ total }} 条记录
        </div>
        <div class="flex items-center space-x-2">
          <button
            @click="prevPage"
            :disabled="page === 1"
            class="px-3 py-1 border border-gray-300 rounded-md text-sm disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-100"
          >
            上一页
          </button>
          <span class="text-sm text-gray-700">第 {{ page }} 页</span>
          <button
            @click="nextPage"
            :disabled="page * pageSize >= total"
            class="px-3 py-1 border border-gray-300 rounded-md text-sm disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-100"
          >
            下一页
          </button>
        </div>
      </div>
    </div>

    <!-- 创建/编辑弹窗 -->
    <div v-if="showModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-md max-h-[90vh] overflow-y-auto">
        <div class="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
          <h3 class="text-lg font-semibold text-gray-900">{{ isEditing ? '编辑转发规则' : '创建转发规则' }}</h3>
          <button @click="closeModal" class="text-gray-400 hover:text-gray-600">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div class="px-6 py-4 space-y-4">
          <!-- Group 选择 -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">转发组</label>
            <select
              v-model="form.group_id"
              :disabled="isEditing"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100"
            >
              <option value="">请选择转发组</option>
              <option v-for="group in groups" :key="group.id" :value="group.id">
                {{ group.name }}
              </option>
            </select>
          </div>

          <!-- Local Port -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">
              Local Port
              <span class="text-gray-400 text-xs">(30000-33000)</span>
            </label>
            <input
              v-model.number="form.local_port"
              type="number"
              :disabled="isEditing"
              min="30000"
              max="33000"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100"
              placeholder="30000"
            />
          </div>

          <!-- Target Host -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Target Host</label>
            <input
              v-model="form.target_host"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="例如: 192.168.1.100"
            />
          </div>

          <!-- Target Port -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Target Port</label>
            <input
              v-model.number="form.target_port"
              type="number"
              min="1"
              max="65535"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="8080"
            />
          </div>

          <!-- Protocol -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Protocol</label>
            <select
              v-model="form.protocol"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="tcp">TCP</option>
            </select>
          </div>
        </div>
        <div class="px-6 py-4 border-t border-gray-200 flex justify-end space-x-3">
          <button
            @click="closeModal"
            class="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50"
          >
            取消
          </button>
          <button
            @click="saveRule"
            :disabled="saving"
            class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
          >
            {{ saving ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 删除确认弹窗 -->
    <div v-if="showDeleteModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-sm mx-4">
        <div class="px-6 py-4">
          <div class="flex items-center space-x-3 mb-4">
            <div class="w-10 h-10 bg-red-100 rounded-full flex items-center justify-center">
              <svg class="w-6 h-6 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            </div>
            <h3 class="text-lg font-semibold text-gray-900">确认删除</h3>
          </div>
          <p class="text-gray-600">
            确定要删除转发规则 <strong>{{ ruleToDelete?.local_port }} → {{ ruleToDelete?.target_host }}:{{ ruleToDelete?.target_port }}</strong> 吗？此操作不可恢复。
          </p>
        </div>
        <div class="px-6 py-4 border-t border-gray-200 flex justify-end space-x-3">
          <button
            @click="showDeleteModal = false"
            class="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50"
          >
            取消
          </button>
          <button
            @click="deleteRule"
            :disabled="deleting"
            class="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50"
          >
            {{ deleting ? '删除中...' : '删除' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 重启成功提示 -->
    <div v-if="restartSuccess" class="fixed bottom-4 right-4 bg-green-50 border border-green-200 rounded-lg p-4 flex items-center space-x-3 shadow-lg">
      <svg class="w-5 h-5 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
      </svg>
      <span class="text-green-700">重启成功</span>
      <button @click="restartSuccess = false" class="text-green-600 hover:text-green-800">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '../api'

interface Group {
  id: number
  name: string
}

interface Host {
  id: number
  name: string
}

interface Rule {
  id: number
  group_id: number
  local_port: number
  target_host: string
  target_port: number
  protocol: string
  status: string
  group?: Group
  active_host?: Host
}

const rules = ref<Rule[]>([])
const groups = ref<Group[]>([])
const loading = ref(false)
const error = ref('')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

const showModal = ref(false)
const isEditing = ref(false)
const saving = ref(false)
const form = ref<{
  group_id: number | string
  local_port: number
  target_host: string
  target_port: number
  protocol: string
}>({
  group_id: '',
  local_port: 30000,
  target_host: '',
  target_port: 80,
  protocol: 'tcp'
})
const editingRuleId = ref<number | null>(null)

const showDeleteModal = ref(false)
const deleting = ref(false)
const ruleToDelete = ref<Rule | null>(null)

const restartingRuleId = ref<number | null>(null)
const restartSuccess = ref(false)

const statusLabels: Record<string, string> = {
  active: '运行中',
  inactive: '已停止'
}

const statusClasses: Record<string, string> = {
  active: 'bg-green-100 text-green-800',
  inactive: 'bg-gray-100 text-gray-800'
}

const getStatusLabel = (status?: string) => {
  return status ? statusLabels[status] || status : '-'
}

const getStatusClass = (status?: string) => {
  return status ? statusClasses[status] || 'bg-gray-100 text-gray-800' : ''
}

const fetchRules = async () => {
  loading.value = true
  error.value = ''
  try {
    const response = await api.get(`/rules?page=${page.value}&page_size=${pageSize.value}`)
    rules.value = response.data.data || []
    total.value = response.data.total || 0
  } catch (err: any) {
    error.value = err.response?.data?.message || '获取转发规则列表失败'
  } finally {
    loading.value = false
  }
}

const fetchGroups = async () => {
  try {
    const response = await api.get('/groups?page=1&page_size=100')
    groups.value = response.data.data || []
  } catch (err: any) {
    console.error('获取转发组列表失败:', err)
  }
}

const prevPage = () => {
  if (page.value > 1) {
    page.value--
    fetchRules()
  }
}

const nextPage = () => {
  if (page.value * pageSize.value < total.value) {
    page.value++
    fetchRules()
  }
}

const openCreateModal = () => {
  isEditing.value = false
  form.value = {
    group_id: 0,
    local_port: 30000,
    target_host: '',
    target_port: 80,
    protocol: 'tcp'
  }
  editingRuleId.value = null
  fetchGroups()
  showModal.value = true
}

const openEditModal = (rule: Rule) => {
  isEditing.value = true
  form.value = {
    group_id: rule.group_id,
    local_port: rule.local_port,
    target_host: rule.target_host,
    target_port: rule.target_port,
    protocol: rule.protocol
  }
  editingRuleId.value = rule.id
  fetchGroups()
  showModal.value = true
}

const closeModal = () => {
  showModal.value = false
}

const validateForm = () => {
  if (!form.value.group_id) {
    error.value = '请选择转发组'
    return false
  }
  if (!form.value.local_port || form.value.local_port < 30000 || form.value.local_port > 33000) {
    error.value = 'Local Port 必须在 30000-33000 范围内'
    return false
  }
  if (!form.value.target_host.trim()) {
    error.value = '请输入 Target Host'
    return false
  }
  if (!form.value.target_port || form.value.target_port < 1 || form.value.target_port > 65535) {
    error.value = 'Target Port 必须在 1-65535 范围内'
    return false
  }
  return true
}

const saveRule = async () => {
  if (!validateForm()) return

  saving.value = true
  try {
    const payload = {
      ...form.value,
      group_id: parseInt(form.value.group_id as string)
    }
    if (isEditing.value && editingRuleId.value) {
      await api.put(`/rules/${editingRuleId.value}`, payload)
    } else {
      await api.post('/rules', payload)
    }
    closeModal()
    fetchRules()
  } catch (err: any) {
    error.value = err.response?.data?.message || '保存失败'
  } finally {
    saving.value = false
  }
}

const confirmDelete = (rule: Rule) => {
  ruleToDelete.value = rule
  showDeleteModal.value = true
}

const deleteRule = async () => {
  if (!ruleToDelete.value) return

  deleting.value = true
  try {
    await api.delete(`/rules/${ruleToDelete.value.id}`)
    showDeleteModal.value = false
    ruleToDelete.value = null
    fetchRules()
  } catch (err: any) {
    error.value = err.response?.data?.message || '删除失败'
  } finally {
    deleting.value = false
  }
}

const restartRule = async (id: number) => {
  restartingRuleId.value = id
  try {
    await api.post(`/rules/${id}/restart`)
    restartSuccess.value = true
    setTimeout(() => {
      restartSuccess.value = false
    }, 3000)
    fetchRules()
  } catch (err: any) {
    error.value = err.response?.data?.message || '重启失败'
  } finally {
    restartingRuleId.value = null
  }
}

onMounted(() => {
  fetchRules()
})
</script>
