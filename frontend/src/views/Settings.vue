<template>
  <div class="min-h-screen bg-gray-50">
    <NavBar />
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div class="mb-8">
        <h1 class="text-3xl font-bold text-gray-900">Settings</h1>
        <p class="mt-2 text-gray-600">Manage your organization settings</p>
      </div>

      <div class="bg-white shadow rounded-lg p-6 mb-6">
        <h2 class="text-xl font-semibold text-gray-900 mb-4">Connected Organizations</h2>
        <div v-if="orgStore.loading" class="text-center py-8 text-gray-500">
          Loading organizations...
        </div>
        <div v-else-if="orgStore.error" class="bg-red-50 border border-red-200 rounded-lg p-4">
          <p class="text-red-800">{{ orgStore.error }}</p>
        </div>
        <div v-else-if="orgStore.organizations.length === 0" class="text-center py-8 text-gray-500">
          No organizations connected
        </div>
        <ul v-else class="space-y-3">
          <li
            v-for="org in orgStore.organizations"
            :key="org.id"
            class="flex items-center justify-between py-3 px-4 border border-gray-200 rounded-lg hover:bg-gray-50 cursor-pointer"
            :class="{ 'bg-blue-50 border-blue-300': orgStore.current?.id === org.id }"
            @click="selectOrganization(org)"
          >
            <div class="flex items-center">
              <div>
                <p class="text-sm font-medium text-gray-900">{{ org.name }}</p>
                <p class="text-xs text-gray-500">{{ org.slug }}</p>
              </div>
            </div>
            <div class="flex items-center space-x-3">
              <span
                :class="[
                  'px-2 py-1 text-xs font-medium rounded capitalize',
                  org.plan === 'free' ? 'bg-gray-100 text-gray-800' :
                  org.plan === 'pro' ? 'bg-blue-100 text-blue-800' :
                  org.plan === 'team' ? 'bg-purple-100 text-purple-800' :
                  'bg-green-100 text-green-800'
                ]"
              >
                {{ org.plan }}
              </span>
              <span
                v-if="orgStore.current?.id === org.id"
                class="px-2 py-1 text-xs font-medium text-blue-600 bg-blue-100 rounded"
              >
                Active
              </span>
            </div>
          </li>
        </ul>
      </div>

      <div class="bg-white shadow rounded-lg p-6 mb-6">
        <h2 class="text-xl font-semibold text-gray-900 mb-4">Current Organization</h2>
        <div v-if="orgStore.current" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700">Name</label>
            <p class="mt-1 text-sm text-gray-900">{{ orgStore.current.name }}</p>
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700">Plan</label>
            <p class="mt-1 text-sm text-gray-900 capitalize">{{ orgStore.current.plan }}</p>
          </div>
        </div>
        <div v-else class="text-gray-500 text-sm">
          No organization selected
        </div>
      </div>

      <!-- Global Rules Editor -->
      <GlobalRulesEditor
        v-if="orgStore.current"
        :rules="globalRules"
        :title="'Organization Global Rules'"
        :description="'Configure global rules that will be applied to all repositories in this organization. Repository-specific rules will override these.'"
        :loading="loadingRules"
        @save="handleSaveRules"
        @cancel="handleCancelRules"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import NavBar from '@/components/NavBar.vue'
import GlobalRulesEditor from '@/components/GlobalRulesEditor.vue'
import { useOrganizationStore } from '@/stores/organization'
import { useAuthStore } from '@/stores/auth'
import { organizationsAPI } from '@/api/organizations'
import type { Organization } from '@/types'
import { ref, watch, onMounted } from 'vue'

const orgStore = useOrganizationStore()
const authStore = useAuthStore()

const globalRules = ref<string[]>([])
const loadingRules = ref(false)

const selectOrganization = async (org: Organization): Promise<void> => {
  orgStore.setOrganization(org)
  await loadGlobalRules(org.id)
}

const loadGlobalRules = async (orgId: string): Promise<void> => {
  loadingRules.value = true
  try {
    globalRules.value = await organizationsAPI.getGlobalRules(orgId)
    if (globalRules.value.length === 0) {
      globalRules.value = ['']
    }
  } catch (error) {
    console.error('Failed to load global rules:', error)
    globalRules.value = ['']
  } finally {
    loadingRules.value = false
  }
}

const handleSaveRules = async (rules: string[]): Promise<void> => {
  if (!orgStore.current) return

  loadingRules.value = true
  try {
    await organizationsAPI.updateGlobalRules(orgStore.current.id, rules)
    globalRules.value = rules
    alert('Global rules saved successfully!')
  } catch (error) {
    console.error('Failed to save global rules:', error)
    alert('Failed to save global rules. Please try again.')
  } finally {
    loadingRules.value = false
  }
}

const handleCancelRules = (): void => {
  // Rules are reset in the component
}

// Watch for org changes
watch(() => orgStore.current, async (newOrg) => {
  if (newOrg) {
    await loadGlobalRules(newOrg.id)
  }
})

onMounted(async () => {
  await orgStore.fetchOrganizations()
  if (orgStore.current) {
    await loadGlobalRules(orgStore.current.id)
  } else if (authStore.orgId) {
    // Try to load rules for current org
    await loadGlobalRules(authStore.orgId)
  }
})
</script>
