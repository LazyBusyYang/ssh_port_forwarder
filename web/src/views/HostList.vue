<template>
  <div class="h-full flex flex-col">
    <!-- 页面标题和操作按钮 -->
    <div class="flex flex-col sm:flex-row justify-between items-start sm:items-center mb-6 gap-4">
      <h1 class="text-xl sm:text-2xl font-bold text-gray-800">SSH Hosts</h1>
      <button
        @click="openCreateModal"
        class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg flex items-center transition-colors whitespace-nowrap"
      >
        <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        新建 Host
      </button>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" class="flex justify-center items-center py-12">
      <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      <span class="ml-3 text-gray-600">加载中...</span>
    </div>

    <!-- 错误提示 -->
    <div v-else-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700 mb-4">
      {{ error }}
    </div>

    <!-- 空状态 -->
    <div v-else-if="hosts.length === 0" class="bg-white rounded-lg shadow p-12 text-center">
      <div class="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center mx-auto mb-4">
        <svg class="w-8 h-8 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
        </svg>
      </div>
      <p class="text-gray-500 mb-4">暂无 SSH Host，点击上方按钮创建</p>
    </div>

    <!-- 主机列表表格 - 添加水平滚动 -->
    <div v-else class="bg-white rounded-lg shadow overflow-hidden flex-1 flex flex-col min-h-0">
      <div class="overflow-x-auto flex-1">
        <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Host</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Port</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Username</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Auth Method</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Weight</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
            <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">操作</th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-for="host in hosts" :key="host.id" class="hover:bg-gray-50">
            <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{{ host.name }}</td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ host.host }}</td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ host.port }}</td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ host.username }}</td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
              <span class="px-2 py-1 bg-gray-100 rounded text-xs">
                {{ host.auth_method === 'password' ? '密码' : '私钥' }}
              </span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ host.weight }}</td>
            <td class="px-6 py-4 whitespace-nowrap">
              <span
                :class="{
                  'px-2 py-1 text-xs rounded-full': true,
                  'bg-green-100 text-green-800': host.health_status === 'healthy',
                  'bg-red-100 text-red-800': host.health_status === 'unhealthy',
                  'bg-gray-100 text-gray-800': host.health_status === 'unknown' || !host.health_status
                }"
              >
                {{ host.health_status === 'healthy' ? '健康' : host.health_status === 'unhealthy' ? '异常' : '未知' }}
              </span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
              <button
                @click="testConnection(host.id)"
                :disabled="testingId === host.id"
                class="text-cyan-600 hover:text-cyan-900 mr-3 disabled:opacity-50"
              >
                {{ testingId === host.id ? '测试中...' : '测试' }}
              </button>
              <button @click="openEditModal(host)" class="text-blue-600 hover:text-blue-900 mr-3">编辑</button>
              <button @click="openCopyModal(host)" class="text-violet-600 hover:text-violet-900 mr-3">复制</button>
              <button @click="confirmDelete(host)" class="text-red-600 hover:text-red-900">删除</button>
            </td>
          </tr>
        </tbody>
        </table>
      </div>

      <!-- 分页 -->
      <div class="bg-gray-50 px-4 sm:px-6 py-3 flex flex-col sm:flex-row items-center justify-between border-t border-gray-200 gap-2">
        <div class="text-sm text-gray-500">
          共 {{ total }} 条记录，第 {{ page }} 页
        </div>
        <div class="flex space-x-2">
          <button
            @click="changePage(page - 1)"
            :disabled="page <= 1"
            class="px-3 py-1 border border-gray-300 rounded text-sm disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-100"
          >
            上一页
          </button>
          <button
            @click="changePage(page + 1)"
            :disabled="page * pageSize >= total"
            class="px-3 py-1 border border-gray-300 rounded text-sm disabled:opacity-50 disabled:cursor-not-allowed hover:bg-gray-100"
          >
            下一页
          </button>
        </div>
      </div>
    </div>

    <!-- 创建/编辑弹窗 - 响应式改进 -->
    <div v-if="showModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-lg max-h-[90vh] overflow-y-auto">
        <div class="px-6 py-4 border-b border-gray-200 flex justify-between items-center">
          <h2 class="text-lg font-semibold text-gray-800">{{ modalTitle }}</h2>
          <button @click="closeModal" class="text-gray-400 hover:text-gray-600">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <form @submit.prevent="saveHost" class="p-6">
          <!-- Name -->
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Name <span class="text-red-500">*</span></label>
            <input
              v-model="form.name"
              type="text"
              required
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="输入主机名称"
            />
          </div>

          <!-- Host -->
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Host <span class="text-red-500">*</span></label>
            <input
              v-model="form.host"
              type="text"
              required
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="例如：192.168.1.100 或 example.com"
            />
          </div>

          <!-- Port -->
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Port <span class="text-red-500">*</span></label>
            <input
              v-model.number="form.port"
              type="number"
              required
              min="1"
              max="65535"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <!-- Username -->
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Username <span class="text-red-500">*</span></label>
            <input
              v-model="form.username"
              type="text"
              required
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="输入用户名"
            />
          </div>

          <!-- Auth Method -->
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">Auth Method <span class="text-red-500">*</span></label>
            <select
              v-model="form.auth_method"
              required
              :disabled="isCopying"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-gray-100"
            >
              <option value="password">密码</option>
              <option value="private_key">私钥</option>
            </select>
            <p v-if="isCopying" class="text-xs text-gray-500 mt-1">与源主机一致；密文在服务端复制，不经浏览器</p>
          </div>

          <!-- Auth Data -->
          <div class="mb-4">
            <label class="block text-sm font-medium text-gray-700 mb-1">
              {{ form.auth_method === 'password' ? 'Password' : 'Private Key' }}
              <span v-if="authDataRequired" class="text-red-500">*</span>
            </label>
            <p v-if="isCopying" class="text-xs text-gray-600 mb-2">
              留空则沿用源主机认证密文（仅服务端复制）；填写则表示用新密码/私钥覆盖副本
            </p>
            <input
              v-if="form.auth_method === 'password'"
              v-model="form.auth_data"
              type="password"
              :required="authDataRequired"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              :placeholder="
                isCopying
                  ? '留空则服务端复制源主机密文'
                  : isEditing && !authDataRequired
                    ? '留空则不修改（认证方式未变更）'
                    : '输入密码'
              "
            />
            <textarea
              v-else
              v-model="form.auth_data"
              :required="authDataRequired"
              rows="4"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono text-sm"
              :placeholder="
                isCopying
                  ? '留空则服务端复制源主机密文'
                  : isEditing && !authDataRequired
                    ? '留空则不修改（认证方式未变更）'
                    : '-----BEGIN OPENSSH PRIVATE KEY-----'
              "
            ></textarea>
          </div>

          <!-- Weight -->
          <div class="mb-6">
            <label class="block text-sm font-medium text-gray-700 mb-1">Weight (1-100)</label>
            <input
              v-model.number="form.weight"
              type="number"
              min="1"
              max="100"
              class="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
            <p class="text-xs text-gray-500 mt-1">权重越高，负载均衡时分配的流量越多</p>
          </div>

          <!-- 按钮 -->
          <div class="flex justify-end space-x-3">
            <button
              type="button"
              @click="closeModal"
              class="px-4 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50 transition-colors"
            >
              取消
            </button>
            <button
              type="submit"
              :disabled="saving"
              class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
            >
              {{ saving ? '保存中...' : '保存' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- 删除确认弹窗 -->
    <div v-if="showDeleteModal" class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-sm mx-4 p-6">
        <div class="flex items-center mb-4">
          <div class="w-10 h-10 bg-red-100 rounded-full flex items-center justify-center mr-3">
            <svg class="w-6 h-6 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <h2 class="text-lg font-semibold text-gray-800">确认删除</h2>
        </div>
        <p class="text-gray-600 mb-6">
          确定要删除主机 <span class="font-semibold">{{ deleteTarget?.name }}</span> 吗？此操作不可恢复。
        </p>
        <div class="flex justify-end space-x-3">
          <button
            @click="showDeleteModal = false"
            class="px-4 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50 transition-colors"
          >
            取消
          </button>
          <button
            @click="deleteHost"
            :disabled="deleting"
            class="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 disabled:opacity-50 transition-colors"
          >
            {{ deleting ? '删除中...' : '删除' }}
          </button>
        </div>
      </div>
    </div>

    <!-- 消息提示 -->
    <div
      v-if="message"
      :class="{
        'fixed bottom-4 right-4 px-6 py-3 rounded-lg shadow-lg transition-all': true,
        'bg-green-500 text-white': messageType === 'success',
        'bg-red-500 text-white': messageType === 'error'
      }"
    >
      {{ message }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import api from '../api'

interface Host {
  id: number
  name: string
  host: string
  port: number
  username: string
  auth_method: 'password' | 'private_key'
  auth_data?: string
  weight: number
  health_status?: 'healthy' | 'unhealthy' | 'unknown'
}

interface HostForm {
  name: string
  host: string
  port: number
  username: string
  auth_method: 'password' | 'private_key'
  auth_data: string
  weight: number
}

const hosts = ref<Host[]>([])
const loading = ref(false)
const error = ref('')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

// 弹窗状态
const showModal = ref(false)
const isEditing = ref(false)
const isCopying = ref(false)
const copySourceId = ref<number | null>(null)
const editingId = ref<number | null>(null)
const saving = ref(false)
const originalAuthMethod = ref<'password' | 'private_key'>('password')

const authDataRequired = computed(() => {
  if (isCopying.value) return false
  if (!isEditing.value) return true
  return form.value.auth_method !== originalAuthMethod.value
})

const modalTitle = computed(() => {
  if (isEditing.value) return '编辑 Host'
  if (isCopying.value) return '复制 Host'
  return '新建 Host'
})

// 删除弹窗状态
const showDeleteModal = ref(false)
const deleteTarget = ref<Host | null>(null)
const deleting = ref(false)

// 测试连接状态
const testingId = ref<number | null>(null)

// 消息提示
const message = ref('')
const messageType = ref<'success' | 'error'>('success')

// 表单数据
const defaultForm: HostForm = {
  name: '',
  host: '',
  port: 22,
  username: '',
  auth_method: 'password',
  auth_data: '',
  weight: 100
}
const form = ref<HostForm>({ ...defaultForm })

// 获取主机列表
const fetchHosts = async () => {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get('/hosts', {
      params: { page: page.value, page_size: pageSize.value }
    })
    if (res.data.code === 0) {
      hosts.value = res.data.data
      total.value = res.data.total
      page.value = res.data.page
      pageSize.value = res.data.page_size
    } else {
      error.value = res.data.message || '获取数据失败'
    }
  } catch (err: any) {
    error.value = err.response?.data?.message || '网络错误，请稍后重试'
  } finally {
    loading.value = false
  }
}

// 分页
const changePage = (newPage: number) => {
  if (newPage < 1) return
  page.value = newPage
  fetchHosts()
}

// 打开创建弹窗
const openCreateModal = () => {
  isEditing.value = false
  isCopying.value = false
  copySourceId.value = null
  editingId.value = null
  form.value = { ...defaultForm }
  showModal.value = true
}

const openEditModal = (host: Host) => {
  isEditing.value = true
  isCopying.value = false
  copySourceId.value = null
  originalAuthMethod.value = host.auth_method
  editingId.value = host.id
  form.value = {
    name: host.name,
    host: host.host,
    port: host.port,
    username: host.username,
    auth_method: host.auth_method,
    auth_data: '',
    weight: host.weight
  }
  showModal.value = true
}

const openCopyModal = (host: Host) => {
  isEditing.value = false
  isCopying.value = true
  copySourceId.value = host.id
  editingId.value = null
  originalAuthMethod.value = host.auth_method
  form.value = {
    name: `${host.name}_copy`,
    host: host.host,
    port: host.port,
    username: host.username,
    auth_method: host.auth_method,
    auth_data: '',
    weight: host.weight
  }
  showModal.value = true
}

const closeModal = () => {
  showModal.value = false
  isCopying.value = false
  copySourceId.value = null
  form.value = { ...defaultForm }
}

// 保存主机
const saveHost = async () => {
  if (authDataRequired.value && !String(form.value.auth_data || '').trim()) {
    showMessage('请填写密码或私钥', 'error')
    return
  }
  saving.value = true
  try {
    if (isCopying.value && copySourceId.value != null) {
      const payload: Record<string, unknown> = {
        name: form.value.name,
        host: form.value.host,
        port: form.value.port,
        username: form.value.username,
        weight: form.value.weight
      }
      if (String(form.value.auth_data || '').trim()) {
        payload.auth_data = form.value.auth_data
      }
      const res = await api.post(`/hosts/${copySourceId.value}/copy`, payload)
      if (res.data.code === 0) {
        showMessage('复制并创建成功', 'success')
        closeModal()
        fetchHosts()
      } else {
        showMessage(res.data.message || '复制失败', 'error')
      }
    } else if (isEditing.value && editingId.value) {
      const payload: Record<string, unknown> = { ...form.value }
      if (!payload.auth_data) {
        delete payload.auth_data
      }
      const res = await api.put(`/hosts/${editingId.value}`, payload)
      if (res.data.code === 0) {
        showMessage('更新成功', 'success')
        closeModal()
        fetchHosts()
      } else {
        showMessage(res.data.message || '更新失败', 'error')
      }
    } else {
      const res = await api.post('/hosts', form.value)
      if (res.data.code === 0) {
        showMessage('创建成功', 'success')
        closeModal()
        fetchHosts()
      } else {
        showMessage(res.data.message || '创建失败', 'error')
      }
    }
  } catch (err: any) {
    showMessage(err.response?.data?.message || '操作失败', 'error')
  } finally {
    saving.value = false
  }
}

// 确认删除
const confirmDelete = (host: Host) => {
  deleteTarget.value = host
  showDeleteModal.value = true
}

// 删除主机
const deleteHost = async () => {
  if (!deleteTarget.value) return
  deleting.value = true
  try {
    const res = await api.delete(`/hosts/${deleteTarget.value.id}`)
    if (res.data.code === 0) {
      showMessage('删除成功', 'success')
      showDeleteModal.value = false
      deleteTarget.value = null
      fetchHosts()
    } else {
      showMessage(res.data.message || '删除失败', 'error')
    }
  } catch (err: any) {
    showMessage(err.response?.data?.message || '删除失败', 'error')
  } finally {
    deleting.value = false
  }
}

// 测试连接
const testConnection = async (id: number) => {
  testingId.value = id
  try {
    const res = await api.post(`/hosts/${id}/test`)
    if (res.data.code === 0) {
      showMessage(res.data.message || '连接成功', 'success')
    } else {
      showMessage(res.data.message || '连接失败', 'error')
    }
  } catch (err: any) {
    showMessage(err.response?.data?.message || '连接测试失败', 'error')
  } finally {
    testingId.value = null
  }
}

// 显示消息
const showMessage = (msg: string, type: 'success' | 'error') => {
  message.value = msg
  messageType.value = type
  setTimeout(() => {
    message.value = ''
  }, 3000)
}

onMounted(() => {
  fetchHosts()
})
</script>
