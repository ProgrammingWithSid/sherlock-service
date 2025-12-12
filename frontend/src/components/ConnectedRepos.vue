<template>
  <div class="bg-white rounded-lg shadow">
    <div class="px-6 py-4 border-b border-gray-200">
      <h2 class="text-lg font-semibold text-gray-900">Connected Repositories</h2>
    </div>
    <div class="px-6 py-4">
      <div v-if="loading" class="text-center py-8 text-gray-500">
        Loading...
      </div>
      <div v-else-if="repos.length === 0" class="text-center py-8 text-gray-500">
        No repositories connected
      </div>
      <ul v-else class="space-y-3">
        <li
          v-for="repo in repos"
          :key="repo.id"
          class="flex items-center justify-between py-2"
        >
          <div class="flex items-center">
            <span
              :class="[
                'w-2 h-2 rounded-full mr-3',
                repo.is_active ? 'bg-green-500' : 'bg-gray-400'
              ]"
            />
            <div>
              <p class="text-sm font-medium text-gray-900">{{ repo.full_name }}</p>
              <p class="text-xs text-gray-500">{{ repo.platform }}</p>
            </div>
          </div>
          <span
            :class="[
              'px-2 py-1 text-xs font-medium rounded',
              repo.is_active ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
            ]"
          >
            {{ repo.is_active ? 'Active' : 'Paused' }}
          </span>
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRepositoriesStore } from '@/stores/repositories'
import type { Repository } from '@/types'
import { onMounted, ref } from 'vue'

const reposStore = useRepositoriesStore()
const repos = ref<Repository[]>([])
const loading = ref(true)

onMounted(async () => {
  try {
    await reposStore.fetchRepositories()
    repos.value = reposStore.activeRepositories.slice(0, 5)
  } catch (err) {
    console.error('Failed to fetch repositories:', err)
  } finally {
    loading.value = false
  }
})
</script>
