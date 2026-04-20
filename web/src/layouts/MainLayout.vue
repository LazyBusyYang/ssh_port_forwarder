<template>
  <div class="flex h-screen bg-gray-100 overflow-hidden">
    <!-- 左侧边栏 - 响应式：小屏幕时隐藏或变窄 -->
    <aside 
      class="bg-gray-900 text-gray-300 flex flex-col flex-shrink-0 transition-all duration-300"
      :class="{ 
        'w-60': !isMobile, 
        'w-16': isMobile && !sidebarCollapsed,
        '-ml-60': isMobile && sidebarCollapsed 
      }"
    >
      <!-- 移动端侧边栏切换按钮 -->
      <button 
        v-if="isMobile"
        @click="toggleSidebar"
        class="absolute -right-10 top-4 bg-gray-900 text-white p-2 rounded-r-lg z-50"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>
      <!-- Logo/标题 -->
      <div class="h-16 flex items-center px-6 border-b border-gray-800 overflow-hidden">
        <h1 class="text-lg font-semibold text-white whitespace-nowrap" :class="{ 'hidden': isMobile && !sidebarCollapsed }">SSH Forwarder</h1>
        <h1 v-if="isMobile && sidebarCollapsed" class="text-lg font-semibold text-white">SF</h1>
      </div>

      <!-- 导航菜单 -->
      <nav class="flex-1 py-4 overflow-y-auto">
        <ul class="space-y-1 px-3">
          <li>
            <router-link
              to="/dashboard"
              class="flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors hover:bg-gray-800 hover:text-white"
              :class="{ 'bg-gray-800 text-white': $route.path === '/dashboard' }"
            >
              <svg class="w-5 h-5 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
              </svg>
              <span :class="{ 'hidden': isMobile && !sidebarCollapsed }">Dashboard</span>
            </router-link>
          </li>
          <li>
            <router-link
              to="/hosts"
              class="flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors hover:bg-gray-800 hover:text-white"
              :class="{ 'bg-gray-800 text-white': $route.path === '/hosts' }"
            >
              <svg class="w-5 h-5" :class="{ 'mr-3': !isMobile || sidebarCollapsed }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
              </svg>
              <span :class="{ 'hidden': isMobile && !sidebarCollapsed }">SSH Hosts</span>
            </router-link>
          </li>
          <li>
            <router-link
              to="/groups"
              class="flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors hover:bg-gray-800 hover:text-white"
              :class="{ 'bg-gray-800 text-white': $route.path === '/groups' }"
            >
              <svg class="w-5 h-5 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
              </svg>
              <span :class="{ 'hidden': isMobile && !sidebarCollapsed }">Forward Groups</span>
            </router-link>
          </li>
          <li>
            <router-link
              to="/rules"
              class="flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors hover:bg-gray-800 hover:text-white"
              :class="{ 'bg-gray-800 text-white': $route.path === '/rules' }"
            >
              <svg class="w-5 h-5" :class="{ 'mr-3': !isMobile || sidebarCollapsed }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
              </svg>
              <span :class="{ 'hidden': isMobile && !sidebarCollapsed }">Forward Rules</span>
            </router-link>
          </li>
          <li>
            <router-link
              to="/health"
              class="flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors hover:bg-gray-800 hover:text-white"
              :class="{ 'bg-gray-800 text-white': $route.path === '/health' }"
            >
              <svg class="w-5 h-5" :class="{ 'mr-3': !isMobile || sidebarCollapsed }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
              </svg>
              <span :class="{ 'hidden': isMobile && !sidebarCollapsed }">Health History</span>
            </router-link>
          </li>
          <li>
            <router-link
              to="/audit-logs"
              class="flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors hover:bg-gray-800 hover:text-white"
              :class="{ 'bg-gray-800 text-white': $route.path === '/audit-logs' }"
            >
              <svg class="w-5 h-5" :class="{ 'mr-3': !isMobile || sidebarCollapsed }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span :class="{ 'hidden': isMobile && !sidebarCollapsed }">Audit Logs</span>
            </router-link>
          </li>
        </ul>
      </nav>

      <!-- 底部信息 -->
      <div class="px-6 py-4 border-t border-gray-800 text-xs text-gray-500 overflow-hidden">
        <span :class="{ 'hidden': isMobile && !sidebarCollapsed }">SSH Port Forwarder v1.0</span>
        <span v-if="isMobile && sidebarCollapsed">v1.0</span>
      </div>
    </aside>

    <!-- 右侧主内容区 - 确保可以滚动 -->
    <div class="flex-1 flex flex-col min-w-0 overflow-hidden">
      <!-- 顶部栏 -->
      <header class="h-16 bg-white border-b border-gray-200 flex items-center justify-between px-4 lg:px-6 flex-shrink-0">
        <div class="text-sm text-gray-500">
          {{ $route.name }}
        </div>
        <div class="flex items-center space-x-4">
          <!-- WebSocket 连接状态 -->
          <div class="flex items-center gap-2 px-3 py-1.5 bg-gray-100 rounded-full">
            <div
              :class="[
                'w-2.5 h-2.5 rounded-full',
                wsConnected ? 'bg-green-500' : 'bg-red-500'
              ]"
            ></div>
            <span class="text-xs text-gray-600">
              {{ wsConnected ? '实时连接' : '连接断开' }}
            </span>
          </div>
          <div class="flex items-center space-x-2">
            <div class="w-8 h-8 bg-blue-600 rounded-full flex items-center justify-center">
              <svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
              </svg>
            </div>
            <span class="text-sm text-gray-700">{{ authStore.username || 'Admin' }}</span>
            <span v-if="authStore.isAdmin" class="px-2 py-0.5 bg-blue-100 text-blue-700 text-xs rounded-full">管理员</span>
          </div>
          <button
            class="text-sm text-red-600 hover:text-red-800 font-medium"
            @click="handleLogout"
          >
            登出
          </button>
        </div>
      </header>

      <!-- 主内容 - 添加水平和垂直滚动 -->
      <main class="flex-1 overflow-auto p-4 lg:p-6 min-w-0">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useStatusWebSocket } from '../composables/useWebSocket'
import { ref, onMounted, onUnmounted } from 'vue'

const router = useRouter()
const authStore = useAuthStore()
const { connected: wsConnected } = useStatusWebSocket()

// 响应式侧边栏状态
const isMobile = ref(false)
const sidebarCollapsed = ref(true)

const checkScreenSize = () => {
  // 竖屏 9:16 或宽度小于 768px 视为移动端
  isMobile.value = window.innerWidth < 768 || window.innerHeight > window.innerWidth
}

const toggleSidebar = () => {
  sidebarCollapsed.value = !sidebarCollapsed.value
}

onMounted(() => {
  checkScreenSize()
  window.addEventListener('resize', checkScreenSize)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkScreenSize)
})

const handleLogout = () => {
  authStore.logout()
  router.push('/login')
}
</script>
