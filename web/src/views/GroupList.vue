<template>
  <div class="h-full flex flex-col space-y-6">
    <!-- 页面标题 -->
    <div class="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
      <h1 class="text-xl sm:text-2xl font-bold text-gray-900">Forward Groups</h1>
      <button
        @click="openCreateModal"
        class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors flex items-center space-x-2 whitespace-nowrap"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        <span>创建转发组</span>
      </button>
    </div>

    <!-- 错误提示 -->
    <div v-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4 flex items-center space-x-3">
      <svg class="w-5 h-5 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <span class="text-red-700">{{ error }}</span>
      <button @click="fetchGroups" class="text-red-600 hover:text-red-800 underline ml-auto">重试</button>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="flex items-center justify-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      <span class="ml-3 text-gray-500">加载中...</span>
    </div>

    <!-- 空状态 -->
    <div v-else-if="groups.length === 0" class="bg-white rounded-lg shadow-sm border border-gray-200 p-12 text-center">
      <svg class="w-16 h-16 text-gray-300 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
      </svg>
      <h3 class="text-lg font-medium text-gray-900 mb-1">暂无转发组</h3>
      <p class="text-gray-500 mb-4">创建第一个转发组来管理 SSH 主机</p>
      <button
        @click="openCreateModal"
        class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
      >
        创建转发组
      </button>
    </div>

    <!-- 表格 - 添加水平滚动 -->
    <div v-else class="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden flex-1 flex flex-col min-h-0">
      <div class="overflow-x-auto flex-1">
        <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Strategy</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Hosts</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Rules</th>
            <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">操作</th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-for="group in groups" :key="group.id" class="hover:bg-gray-50">
            <td class="px-6 py-4 whitespace-nowrap">
              <button
                @click="viewGroupDetail(group.id)"
                class="text-blue-600 hover:text-blue-800 font-medium"
              >
                {{ group.name }}
              </button>
            </td>
            <td class="px-6 py-4 whitespace-nowrap">
              <span class="px-2 py-1 text-xs rounded-full" :class="getStrategyClass(group.strategy)">
                {{ getStrategyLabel(group.strategy) }}
              </span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-gray-700">
              {{ group.hosts?.length || 0 }}
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-gray-700">
              {{ group.rules?.length || 0 }}
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-right space-x-2">
              <button
                @click="viewGroupDetail(group.id)"
                class="text-blue-600 hover:text-blue-800 text-sm"
              >
                详情
              </button>
              <button
                @click="openEditModal(group)"
                class="text-indigo-600 hover:text-indigo-800 text-sm"
              >
                编辑
              </button>
              <button
                @click="confirmDelete(group)"
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
          <h3 class="text-lg font-semibold text-gray-900">{{ isEditing ? '编辑转发组' : '创建转发组' }}</h3>
          <button @click="closeModal" class="text-gray-400 hover:text-gray-600">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div class="px-6 py-4 space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">名称</label>
            <input
              v-model="form.name"
              type="text"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="输入转发组名称"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">负载均衡策略</label>
            <select
              v-model="form.strategy"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="round_robin">轮询 (Round Robin)</option>
              <option value="least_rules">最少规则 (Least Rules)</option>
              <option value="weighted">加权 (Weighted)</option>
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
            @click="saveGroup"
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
            确定要删除转发组 <strong>{{ groupToDelete?.name }}</strong> 吗？此操作不可恢复。
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
            @click="deleteGroup"
            :disabled="deleting"
            class="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50"
          >
            {{ deleting ? '删除中...' : '删除' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 详情弹窗 -->
    <div v-if="showDetailModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-2xl mx-4 max-h-[90vh] overflow-y-auto">
        <div class="px-6 py-4 border-b border-gray-200 flex items-center justify-between sticky top-0 bg-white">
          <h3 class="text-lg font-semibold text-gray-900">转发组详情: {{ groupDetail?.name }}</h3>
          <button @click="closeDetailModal" class="text-gray-400 hover:text-gray-600">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div class="px-6 py-4 space-y-6">
          <!-- 基本信息 -->
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="text-sm text-gray-500">策略</label>
              <p class="font-medium">{{ getStrategyLabel(groupDetail?.strategy) }}</p>
            </div>
            <div>
              <label class="text-sm text-gray-500">规则数量</label>
              <p class="font-medium">{{ groupDetail?.rules?.length || 0 }}</p>
            </div>
          </div>

          <!-- Host 管理 -->
          <div>
            <div class="flex items-center justify-between mb-3">
              <h4 class="font-medium text-gray-900">关联主机</h4>
              <button
                @click="openAddHostModal"
                class="px-3 py-1.5 bg-blue-600 text-white text-sm rounded-lg hover:bg-blue-700"
              >
                添加 Host
              </button>
            </div>
            <div v-if="groupDetail?.hosts?.length === 0" class="text-gray-500 text-sm py-4 text-center bg-gray-50 rounded-lg">
              暂无关联主机
            </div>
            <table v-else class="min-w-full divide-y divide-gray-200 border rounded-lg">
              <thead class="bg-gray-50">
                <tr>
                  <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">名称</th>
                  <th class="px-4 py-2 text-left text-xs font-medium text-gray-500 uppercase">地址</th>
                  <th class="px-4 py-2 text-right text-xs font-medium text-gray-500 uppercase">操作</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-200">
                <tr v-for="host in groupDetail?.hosts" :key="host.id">
                  <td class="px-4 py-2 text-sm">{{ host.name }}</td>
                  <td class="px-4 py-2 text-sm text-gray-500">{{ host.host }}:{{ host.port }}</td>
                  <td class="px-4 py-2 text-right">
                    <button
                      @click="removeHost(host.id)"
                      class="text-red-600 hover:text-red-800 text-sm"
                    >
                      移除
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>

    <!-- 添加 Host 弹窗 -->
    <div v-if="showAddHostModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-md mx-4">
        <div class="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
          <h3 class="text-lg font-semibold text-gray-900">添加 Host</h3>
          <button @click="closeAddHostModal" class="text-gray-400 hover:text-gray-600">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div class="px-6 py-4">
          <label class="block text-sm font-medium text-gray-700 mb-1">选择 Host</label>
          <select
            v-model="selectedHostId"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="">请选择</option>
            <option v-for="host in availableHosts" :key="host.id" :value="host.id">
              {{ host.name }} ({{ host.host }}:{{ host.port }})
            </option>
          </select>
        </div>
        <div class="px-6 py-4 border-t border-gray-200 flex justify-end space-x-3">
          <button
            @click="closeAddHostModal"
            class="px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50"
          >
            取消
          </button>
          <button
            @click="addHost"
            :disabled="!selectedHostId || addingHost"
            class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
          >
            {{ addingHost ? '添加中...' : '添加' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '../api'

interface Host {
  id: number
  name: string
  host: string
  port: number
}

interface Rule {
  id: number
  local_port: number
  target_host: string
  target_port: number
}

interface Group {
  id: number
  name: string
  strategy: string
  hosts?: Host[]
  rules?: Rule[]
}

const groups = ref<Group[]>([])
const loading = ref(false)
const error = ref('')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

const showModal = ref(false)
const isEditing = ref(false)
const saving = ref(false)
const form = ref({ name: '', strategy: 'round_robin' })
const editingGroupId = ref<number | null>(null)

const showDeleteModal = ref(false)
const deleting = ref(false)
const groupToDelete = ref<Group | null>(null)

const showDetailModal = ref(false)
const groupDetail = ref<Group | null>(null)

const showAddHostModal = ref(false)
const availableHosts = ref<Host[]>([])
const selectedHostId = ref('')
const addingHost = ref(false)

const strategyLabels: Record<string, string> = {
  round_robin: '轮询',
  least_rules: '最少规则',
  weighted: '加权'
}

const strategyClasses: Record<string, string> = {
  round_robin: 'bg-blue-100 text-blue-800',
  least_rules: 'bg-green-100 text-green-800',
  weighted: 'bg-purple-100 text-purple-800'
}

const getStrategyLabel = (strategy?: string) => {
  return strategy ? strategyLabels[strategy] || strategy : '-'
}

const getStrategyClass = (strategy?: string) => {
  return strategy ? strategyClasses[strategy] || 'bg-gray-100 text-gray-800' : ''
}

const fetchGroups = async () => {
  loading.value = true
  error.value = ''
  try {
    const response = await api.get(`/groups?page=${page.value}&page_size=${pageSize.value}`)
    groups.value = response.data.data || []
    total.value = response.data.total || 0
  } catch (err: any) {
    error.value = err.response?.data?.message || '获取转发组列表失败'
  } finally {
    loading.value = false
  }
}

const prevPage = () => {
  if (page.value > 1) {
    page.value--
    fetchGroups()
  }
}

const nextPage = () => {
  if (page.value * pageSize.value < total.value) {
    page.value++
    fetchGroups()
  }
}

const openCreateModal = () => {
  isEditing.value = false
  form.value = { name: '', strategy: 'round_robin' }
  editingGroupId.value = null
  showModal.value = true
}

const openEditModal = (group: Group) => {
  isEditing.value = true
  form.value = { name: group.name, strategy: group.strategy }
  editingGroupId.value = group.id
  showModal.value = true
}

const closeModal = () => {
  showModal.value = false
}

const saveGroup = async () => {
  if (!form.value.name.trim()) {
    error.value = '请输入转发组名称'
    return
  }

  saving.value = true
  try {
    if (isEditing.value && editingGroupId.value) {
      await api.put(`/groups/${editingGroupId.value}`, form.value)
    } else {
      await api.post('/groups', form.value)
    }
    closeModal()
    fetchGroups()
  } catch (err: any) {
    error.value = err.response?.data?.message || '保存失败'
  } finally {
    saving.value = false
  }
}

const confirmDelete = (group: Group) => {
  groupToDelete.value = group
  showDeleteModal.value = true
}

const deleteGroup = async () => {
  if (!groupToDelete.value) return

  deleting.value = true
  try {
    await api.delete(`/groups/${groupToDelete.value.id}`)
    showDeleteModal.value = false
    groupToDelete.value = null
    fetchGroups()
  } catch (err: any) {
    error.value = err.response?.data?.message || '删除失败'
  } finally {
    deleting.value = false
  }
}

const viewGroupDetail = async (id: number) => {
  try {
    const response = await api.get(`/groups/${id}`)
    groupDetail.value = response.data.data
    showDetailModal.value = true
  } catch (err: any) {
    error.value = err.response?.data?.message || '获取详情失败'
  }
}

const closeDetailModal = () => {
  showDetailModal.value = false
  groupDetail.value = null
}

const openAddHostModal = async () => {
  selectedHostId.value = ''
  try {
    const response = await api.get('/hosts?page=1&page_size=100')
    const allHosts = response.data.data || []
    const existingHostIds = new Set(groupDetail.value?.hosts?.map(h => h.id) || [])
    availableHosts.value = allHosts.filter((h: Host) => !existingHostIds.has(h.id))
    showAddHostModal.value = true
  } catch (err: any) {
    error.value = err.response?.data?.message || '获取 Host 列表失败'
  }
}

const closeAddHostModal = () => {
  showAddHostModal.value = false
  selectedHostId.value = ''
}

const addHost = async () => {
  if (!selectedHostId.value || !groupDetail.value) return

  addingHost.value = true
  try {
    await api.post(`/groups/${groupDetail.value.id}/hosts`, { host_id: parseInt(selectedHostId.value) })
    closeAddHostModal()
    viewGroupDetail(groupDetail.value.id)
  } catch (err: any) {
    error.value = err.response?.data?.message || '添加 Host 失败'
  } finally {
    addingHost.value = false
  }
}

const removeHost = async (hostId: number) => {
  if (!groupDetail.value) return

  try {
    await api.delete(`/groups/${groupDetail.value.id}/hosts/${hostId}`)
    viewGroupDetail(groupDetail.value.id)
  } catch (err: any) {
    error.value = err.response?.data?.message || '移除 Host 失败'
  }
}

onMounted(() => {
  fetchGroups()
})
</script>
