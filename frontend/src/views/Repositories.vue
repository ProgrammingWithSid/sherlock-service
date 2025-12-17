<template>
  <div class="min-h-screen bg-gray-50">
    <NavBar />
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div class="mb-8 flex justify-between items-center">
        <div>
          <h1 class="text-3xl font-bold text-gray-900">Repositories</h1>
          <p class="mt-2 text-gray-600">Manage your connected repositories</p>
        </div>
        <button
          @click="showConnectModal = true"
          class="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700"
        >
          Add Repository
        </button>
      </div>

      <ConnectRepoModal
        :show="showConnectModal"
        @close="showConnectModal = false"
        @connected="handleRepoConnected"
      />

      <RepoConfigModal
        :show="showConfigModal"
        :repo="selectedRepo"
        @close="showConfigModal = false"
        @saved="handleConfigSaved"
      />

      <div v-if="reposStore.loading" class="text-center py-12">
        <p class="text-gray-500">Loading repositories...</p>
      </div>

      <div v-else-if="reposStore.error" class="bg-red-50 border border-red-200 rounded-lg p-4">
        <p class="text-red-800">{{ reposStore.error }}</p>
      </div>

      <div v-else-if="reposStore.items.length === 0" class="text-center py-12">
        <p class="text-gray-500 mb-4">No repositories connected</p>
        <button class="bg-blue-600 text-white px-4 py-2 rounded-lg hover:bg-blue-700">
          Connect Your First Repository
        </button>
      </div>

      <div v-else class="bg-white shadow rounded-lg overflow-hidden">
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Repository
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Platform
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Status
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Created
              </th>
              <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-gray-200">
            <tr v-for="repo in reposStore.items" :key="repo.id">
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm font-medium text-gray-900">{{ repo.full_name }}</div>
                <div v-if="repo.is_private" class="text-xs text-gray-500">Private</div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <span class="px-2 py-1 text-xs font-medium rounded bg-gray-100 text-gray-800">
                  {{ repo.platform }}
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <span
                  :class="[
                    'px-2 py-1 text-xs font-medium rounded',
                    repo.is_active ? 'bg-green-100 text-green-800' : 'bg-gray-100 text-gray-800'
                  ]"
                >
                  {{ repo.is_active ? 'Active' : 'Paused' }}
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                {{ formatDate(repo.created_at) }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <button
                  @click="handleToggleActive(repo.id, repo.is_active)"
                  :class="[
                    'mr-4',
                    repo.is_active ? 'text-yellow-600 hover:text-yellow-900' : 'text-green-600 hover:text-green-900'
                  ]"
                >
                  {{ repo.is_active ? 'Pause' : 'Activate' }}
                </button>
                <button
                  @click="handleConfigure(repo)"
                  class="text-blue-600 hover:text-blue-900 mr-4"
                >
                  Configure
                </button>
                <button class="text-red-600 hover:text-red-900">Remove</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import ConnectRepoModal from '@/components/ConnectRepoModal.vue'
import NavBar from '@/components/NavBar.vue'
import RepoConfigModal from '@/components/RepoConfigModal.vue'
import { useRepositoriesStore } from '@/stores/repositories'
import type { Repository } from '@/types'
import { onMounted, ref } from 'vue'

const reposStore = useRepositoriesStore()
const showConnectModal = ref(false)
const showConfigModal = ref(false)
const selectedRepo = ref<Repository | null>(null)

const formatDate = (dateString: string): string => {
  const date = new Date(dateString)
  return date.toLocaleDateString()
}

const handleToggleActive = async (repoId: string, currentStatus: boolean): Promise<void> => {
  await reposStore.setActive(repoId, !currentStatus)
}

const handleRepoConnected = (): void => {
  reposStore.fetchRepositories()
}

const handleConfigure = (repo: Repository): void => {
  selectedRepo.value = repo
  showConfigModal.value = true
}

const handleConfigSaved = (): void => {
  reposStore.fetchRepositories()
}

onMounted(() => {
  reposStore.fetchRepositories()
})
</script>
