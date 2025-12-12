<template>
  <div class="min-h-screen bg-gray-50 flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
    <div class="max-w-md w-full space-y-8">
      <div>
        <h2 class="mt-6 text-center text-3xl font-extrabold text-gray-900">
          Sign in to Code-Sherlock
        </h2>
        <p class="mt-2 text-center text-sm text-gray-600">
          Connect your GitHub or GitLab account to get started
        </p>
      </div>

      <div class="mt-8 space-y-6">
        <div v-if="error" class="bg-red-50 border border-red-200 rounded-md p-4">
          <p class="text-sm text-red-800">{{ error }}</p>
        </div>

        <div class="space-y-4">
          <button
            @click="loginWithGitHub"
            :disabled="loading"
            class="w-full flex items-center justify-center px-4 py-3 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-gray-900 hover:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-gray-500 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <svg class="w-5 h-5 mr-2" fill="currentColor" viewBox="0 0 20 20">
              <path
                fill-rule="evenodd"
                d="M10 0C4.477 0 0 4.484 0 10.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0110 4.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.203 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.942.359.31.678.921.678 1.856 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0020 10.017C20 4.484 15.522 0 10 0z"
                clip-rule="evenodd"
              />
            </svg>
            {{ loading ? 'Connecting...' : 'Continue with GitHub' }}
          </button>

          <button
            @click="loginWithGitLab"
            :disabled="loading"
            class="w-full flex items-center justify-center px-4 py-3 border border-gray-300 rounded-md shadow-sm text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <svg class="w-5 h-5 mr-2" fill="currentColor" viewBox="0 0 20 20">
              <path
                d="M18.384 8.729h-1.568V5.771c0-2.299-1.832-4.161-4.161-4.161H5.771c-2.299 0-4.161 1.832-4.161 4.161v1.568H.231a.231.231 0 00-.231.231v2.078a.231.231 0 00.231.231h1.379v1.568c0 2.299 1.832 4.161 4.161 4.161h6.884c2.299 0 4.161-1.832 4.161-4.161v-1.568h1.568a.231.231 0 00.231-.231V8.96a.231.231 0 00-.231-.231zM5.771 3.206h6.884c1.416 0 2.565 1.149 2.565 2.565v1.568H3.206V5.771c0-1.416 1.149-2.565 2.565-2.565zm8.449 11.588H5.771c-1.416 0-2.565-1.149-2.565-2.565V9.661h11.588v4.568c0 1.416-1.149 2.565-2.565 2.565z"
              />
            </svg>
            {{ loading ? 'Connecting...' : 'Continue with GitLab' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import { ref } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const authStore = useAuthStore()
const loading = ref(false)
const error = ref<string | null>(null)

const loginWithGitHub = async (): Promise<void> => {
  loading.value = true
  error.value = null

  try {
    const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:3000'
    const redirectUri = `${window.location.origin}/auth/callback`
    window.location.href = `${apiBaseUrl}/api/v1/auth/github?redirect_uri=${encodeURIComponent(redirectUri)}`
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to initiate GitHub login'
    loading.value = false
  }
}

const loginWithGitLab = async (): Promise<void> => {
  loading.value = true
  error.value = null

  try {
    const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:3000'
    const redirectUri = `${window.location.origin}/auth/callback`
    window.location.href = `${apiBaseUrl}/api/v1/auth/gitlab?redirect_uri=${encodeURIComponent(redirectUri)}`
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to initiate GitLab login'
    loading.value = false
  }
}
</script>
