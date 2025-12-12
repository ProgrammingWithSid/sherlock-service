<template>
  <div
    v-if="show"
    class="fixed inset-0 bg-gray-600 bg-opacity-50 overflow-y-auto h-full w-full z-50"
    @click.self="close"
  >
    <div class="relative top-20 mx-auto p-5 border w-96 shadow-lg rounded-md bg-white">
      <div class="mt-3">
        <h3 class="text-lg font-medium text-gray-900 mb-4">Connect Repository</h3>

        <div class="mb-4">
          <label class="block text-sm font-medium text-gray-700 mb-2">
            Platform
          </label>
          <select
            v-model="platform"
            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="github">GitHub</option>
            <option value="gitlab">GitLab</option>
          </select>
        </div>

        <div class="mb-4">
          <label class="block text-sm font-medium text-gray-700 mb-2">
            Repository URL
          </label>
          <input
            v-model="repoUrl"
            type="text"
            placeholder="https://github.com/owner/repo"
            class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
        </div>

        <div v-if="error" class="mb-4 p-3 bg-red-50 border border-red-200 rounded-md">
          <p class="text-sm text-red-800">{{ error }}</p>
        </div>

        <div class="flex justify-end space-x-3">
          <button
            @click="close"
            class="px-4 py-2 bg-gray-200 text-gray-800 rounded-md hover:bg-gray-300"
          >
            Cancel
          </button>
          <button
            @click="handleConnect"
            :disabled="loading || !repoUrl"
            class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {{ loading ? 'Connecting...' : 'Connect' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { repositoriesAPI } from '@/api/repositories'
import { useRepositoriesStore } from '@/stores/repositories'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  close: []
  connected: []
}>()

const reposStore = useRepositoriesStore()
const platform = ref<'github' | 'gitlab'>('github')
const repoUrl = ref('')
const loading = ref(false)
const error = ref<string | null>(null)

const close = (): void => {
  emit('close')
  repoUrl.value = ''
  error.value = null
}

const handleConnect = async (): Promise<void> => {
  if (!repoUrl.value) {
    error.value = 'Please enter a repository URL'
    return
  }

  loading.value = true
  error.value = null

  try {
    // Parse repository URL
    const url = new URL(repoUrl.value)
    const parts = url.pathname.split('/').filter((p) => p)

    if (parts.length < 2) {
      throw new Error('Invalid repository URL format')
    }

    const owner = parts[0]
    const repo = parts[1]

    // Call API to connect repository
    await repositoriesAPI.connect(platform.value, owner, repo)

    emit('connected')
    close()
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to connect repository'
  } finally {
    loading.value = false
  }
}
</script>
