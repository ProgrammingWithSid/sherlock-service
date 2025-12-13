<template>
  <nav class="bg-white shadow-sm border-b border-gray-200">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="flex justify-between h-16">
        <div class="flex">
          <div class="flex-shrink-0 flex items-center">
            <h1 class="text-xl font-bold text-gray-900">Code-Sherlock</h1>
          </div>
          <div class="hidden sm:ml-6 sm:flex sm:space-x-8">
            <router-link
              v-for="item in navItems"
              :key="item.path"
              :to="item.path"
              class="border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700 inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium"
              active-class="border-blue-500 text-gray-900"
            >
              {{ item.name }}
            </router-link>
          </div>
        </div>
        <div class="flex items-center space-x-4">
          <span class="text-sm text-gray-700">{{ authStore.user?.name || 'Organization' }}</span>
          <button
            @click="handleLogout"
            class="text-sm text-gray-500 hover:text-gray-700"
          >
            Logout
          </button>
        </div>
      </div>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'
import { computed } from 'vue'

const router = useRouter()
const authStore = useAuthStore()

const handleLogout = async (): Promise<void> => {
  authStore.logout()
  router.push('/login')
}

interface NavItem {
  name: string
  path: string
}

const navItems = computed<NavItem[]>(() => {
  const items: NavItem[] = [
    { name: 'Dashboard', path: authStore.isSuperAdmin ? '/admin' : '/' },
    { name: 'Repositories', path: '/repositories' },
    { name: 'Reviews', path: '/reviews' },
    { name: 'Settings', path: '/settings' },
  ]
  return items
})
</script>
