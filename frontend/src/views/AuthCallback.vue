<template>
  <div class="min-h-screen bg-gray-50 flex items-center justify-center">
    <div class="text-center">
      <div v-if="loading" class="space-y-4">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
        <p class="text-gray-600">Completing authentication...</p>
      </div>
      <div v-else-if="error" class="space-y-4">
        <div class="text-red-600 text-lg font-semibold">Authentication Failed</div>
        <p class="text-gray-600">{{ error }}</p>
        <router-link
          to="/login"
          class="inline-block px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
        >
          Try Again
        </router-link>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const loading = ref(true)
const error = ref<string | null>(null)

onMounted(async () => {
  try {
    const token = route.query.token as string
    const orgId = route.query.org_id as string

    if (!token || !orgId) {
      error.value = 'Missing authentication parameters'
      loading.value = false
      return
    }

    // Store auth info
    localStorage.setItem('session_token', token)
    localStorage.setItem('org_id', orgId)

    // Update auth store
    await authStore.setAuth(token, orgId)
    await authStore.fetchCurrentUser()

    // Redirect to dashboard
    router.push('/')
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Authentication failed'
    loading.value = false
  }
})
</script>


