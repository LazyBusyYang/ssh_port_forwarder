import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  { path: '/login', name: 'Login', component: () => import('../views/Login.vue') },
  {
    path: '/',
    component: () => import('../layouts/MainLayout.vue'),
    children: [
      { path: '', redirect: '/dashboard' },
      { path: 'dashboard', name: 'Dashboard', component: () => import('../views/Dashboard.vue') },
      { path: 'hosts', name: 'Hosts', component: () => import('../views/HostList.vue') },
      { path: 'groups', name: 'Groups', component: () => import('../views/GroupList.vue') },
      { path: 'rules', name: 'Rules', component: () => import('../views/RuleList.vue') },
      { path: 'health', name: 'Health', component: () => import('../views/HealthHistory.vue') },
      { path: 'audit-logs', name: 'AuditLogs', component: () => import('../views/AuditLogs.vue') },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

// 路由守卫
router.beforeEach((to, _from, next) => {
  const authStore = useAuthStore()
  if (to.path !== '/login' && !authStore.isAuthenticated) {
    next('/login')
  } else if (to.path === '/login' && authStore.isAuthenticated) {
    next('/dashboard')
  } else {
    next()
  }
})

export default router
