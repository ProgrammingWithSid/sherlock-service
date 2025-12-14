<template>
  <div
    v-if="show"
    class="fixed inset-0 bg-gray-600 bg-opacity-50 overflow-y-auto h-full w-full z-50"
    @click.self="close"
  >
    <div class="relative top-10 mx-auto p-6 border w-full max-w-4xl shadow-lg rounded-md bg-white my-10">
      <div class="mb-6">
        <h3 class="text-2xl font-semibold text-gray-900 mb-2">
          Configure Repository: {{ repo?.full_name }}
        </h3>
        <p class="text-sm text-gray-600">
          Repository-specific rules override organization global rules
        </p>
      </div>

      <div v-if="loading" class="text-center py-12">
        <p class="text-gray-500">Loading configuration...</p>
      </div>

      <div v-else class="space-y-6">
        <!-- Organization Global Rules Info -->
        <div v-if="orgRules.length > 0" class="bg-blue-50 border border-blue-200 rounded-lg p-4">
          <div class="flex items-start">
            <div class="flex-shrink-0">
              <svg class="h-5 w-5 text-blue-400" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z" clip-rule="evenodd" />
              </svg>
            </div>
            <div class="ml-3 flex-1">
              <h4 class="text-sm font-medium text-blue-800">Organization Global Rules</h4>
              <p class="mt-1 text-sm text-blue-700">
                These rules are inherited from your organization settings. You can override them below with repository-specific rules.
              </p>
              <div class="mt-3 space-y-2">
                <div
                  v-for="(rule, index) in orgRules"
                  :key="index"
                  class="text-sm text-blue-800 bg-blue-100 px-3 py-2 rounded"
                >
                  {{ rule }}
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Repository Rules Editor -->
        <GlobalRulesEditor
          :rules="repoRules"
          :org-rules="orgRules"
          title="Repository-Specific Rules"
          description="Configure rules that override organization global rules for this repository. Leave empty to use organization rules only."
          :show-org-info="true"
          :loading="false"
          @save="handleSaveRules"
          @cancel="handleCancelRules"
        />

        <!-- Other Config Options (placeholder for future) -->
        <div class="border-t pt-6">
          <h4 class="text-lg font-medium text-gray-900 mb-4">Other Settings</h4>
          <p class="text-sm text-gray-500">
            Additional repository configuration options coming soon...
          </p>
        </div>
      </div>

      <!-- Footer Actions -->
      <div class="mt-6 flex justify-end space-x-3 border-t pt-4">
        <button
          @click="close"
          class="px-4 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50"
        >
          Close
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import GlobalRulesEditor from './GlobalRulesEditor.vue'
import { repositoriesAPI } from '@/api/repositories'
import { organizationsAPI } from '@/api/organizations'
import { useAuthStore } from '@/stores/auth'
import type { Repository, RepoConfig } from '@/types'

const props = defineProps<{
  show: boolean
  repo: Repository | null
}>()

const emit = defineEmits<{
  close: []
  saved: []
}>()

const authStore = useAuthStore()

const loading = ref(false)
const orgRules = ref<string[]>([])
const repoRules = ref<string[]>([])
const repoConfig = ref<RepoConfig | null>(null)

const loadConfig = async (): Promise<void> => {
  if (!props.repo) return

  loading.value = true
  try {
    // Load organization global rules
    if (authStore.orgId) {
      try {
        orgRules.value = await organizationsAPI.getGlobalRules(authStore.orgId)
      } catch (error) {
        console.warn('Failed to load org rules:', error)
        orgRules.value = []
      }
    }

    // Load repository config
    const [owner, repo] = props.repo.full_name.split('/')
    try {
      repoConfig.value = await repositoriesAPI.getConfig(owner, repo)
      repoRules.value = repoConfig.value.rules || []
      if (repoRules.value.length === 0) {
        repoRules.value = ['']
      }
    } catch (error) {
      console.warn('Failed to load repo config:', error)
      repoRules.value = ['']
    }
  } finally {
    loading.value = false
  }
}

const handleSaveRules = async (rules: string[]): Promise<void> => {
  if (!props.repo || !repoConfig.value) return

  loading.value = true
  try {
    const [owner, repo] = props.repo.full_name.split('/')
    
    // Update repo config with new rules
    const updatedConfig: RepoConfig = {
      ...repoConfig.value,
      rules: rules.length > 0 ? rules : undefined, // Remove if empty
    }

    await repositoriesAPI.updateConfig(owner, repo, updatedConfig)
    repoRules.value = rules
    emit('saved')
    alert('Repository rules saved successfully!')
  } catch (error) {
    console.error('Failed to save repo rules:', error)
    alert('Failed to save repository rules. Please try again.')
  } finally {
    loading.value = false
  }
}

const handleCancelRules = (): void => {
  // Reset to original rules
  repoRules.value = repoConfig.value?.rules || ['']
}

const close = (): void => {
  emit('close')
}

// Watch for repo changes
watch(() => props.repo, async (newRepo) => {
  if (newRepo && props.show) {
    await loadConfig()
  }
})

watch(() => props.show, async (isShow) => {
  if (isShow && props.repo) {
    await loadConfig()
  }
})

onMounted(async () => {
  if (props.show && props.repo) {
    await loadConfig()
  }
})
</script>

