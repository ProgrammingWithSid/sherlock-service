<template>
  <div class="min-h-screen bg-gray-50 flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
    <div class="max-w-md w-full space-y-8">
      <div>
        <h2 class="mt-6 text-center text-3xl font-extrabold text-gray-900">
          Sign in to Code-Sherlock
        </h2>
        <p class="mt-2 text-center text-sm text-gray-600">
          Sign in to your account
        </p>
      </div>

      <form class="mt-8 space-y-6" @submit.prevent="handleLogin">
        <div v-if="error" class="bg-red-50 border border-red-200 rounded-md p-4">
          <p class="text-sm text-red-800">{{ error }}</p>
        </div>

        <div class="space-y-4">
          <div>
            <label for="email" class="block text-sm font-medium text-gray-700">Email</label>
            <input
              id="email"
              v-model="form.email"
              type="email"
              required
              class="mt-1 appearance-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm"
              placeholder="you@example.com"
            />
          </div>

          <div>
            <label for="password" class="block text-sm font-medium text-gray-700">Password</label>
            <input
              id="password"
              v-model="form.password"
              type="password"
              required
              class="mt-1 appearance-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm"
              placeholder="••••••••"
            />
          </div>
        </div>

        <div>
          <button
            type="submit"
            :disabled="loading"
            class="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {{ loading ? 'Signing in...' : 'Sign in' }}
          </button>
        </div>


        <div class="text-center">
          <router-link
            to="/signup"
            class="text-sm text-blue-600 hover:text-blue-500"
          >
            Don't have an account? Sign up
          </router-link>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { authAPI } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import { useOrganizationStore } from '@/stores/organization'
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const authStore = useAuthStore()
const orgStore = useOrganizationStore()
const loading = ref(false)
const error = ref<string | null>(null)

const form = ref({
  email: '',
  password: '',
})

const handleLogin = async (): Promise<void> => {
  loading.value = true
  error.value = null

  try {
    const response = await authAPI.login({
      email: form.value.email,
      password: form.value.password,
    })

    // Set auth
    await authStore.setAuth(response.token, response.user.org_id || '')
    authStore.user = {
      org_id: response.user.org_id || '',
      name: response.user.name,
      plan: response.organization?.plan || 'free',
    }

    // Set organization if available
    if (response.organization) {
      orgStore.setOrganization({
        id: response.organization.id,
        name: response.organization.name,
        slug: response.organization.slug,
        plan: response.organization.plan,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      })
    }

    // Redirect based on role
    if (response.user.role === 'super_admin') {
      router.push('/admin')
    } else {
      router.push('/')
    }
  } catch (err: any) {
    error.value = err.response?.data?.error || err.message || 'Failed to sign in'
  } finally {
    loading.value = false
  }
}

</script>
