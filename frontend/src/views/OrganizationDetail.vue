<template>
  <div class="min-h-screen bg-gray-50">
    <NavBar />
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div class="mb-6">
        <button
          @click="$router.push('/admin')"
          class="text-blue-600 hover:text-blue-800 mb-4 flex items-center"
        >
          <svg class="w-4 h-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
          </svg>
          Back to Admin Dashboard
        </button>
        <h1 class="text-3xl font-bold text-gray-900">{{ organization?.name || 'Organization' }}</h1>
        <p class="mt-2 text-gray-600">{{ organization?.slug }}</p>
      </div>

      <div v-if="loading" class="text-center py-12">
        <p class="text-gray-500">Loading organization details...</p>
      </div>

      <div v-else-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4">
        <p class="text-red-800">{{ error }}</p>
      </div>

      <div v-else-if="organization" class="space-y-6">
        <!-- Organization Info Card -->
        <div class="bg-white shadow rounded-lg overflow-hidden">
          <div class="px-6 py-4 border-b border-gray-200">
            <h2 class="text-lg font-semibold text-gray-900">Organization Information</h2>
          </div>
          <div class="px-6 py-4">
            <dl class="grid grid-cols-1 gap-x-4 gap-y-6 sm:grid-cols-2">
              <div>
                <dt class="text-sm font-medium text-gray-500">Name</dt>
                <dd class="mt-1 text-sm text-gray-900">{{ organization.name }}</dd>
              </div>
              <div>
                <dt class="text-sm font-medium text-gray-500">Slug</dt>
                <dd class="mt-1 text-sm text-gray-900">{{ organization.slug }}</dd>
              </div>
              <div>
                <dt class="text-sm font-medium text-gray-500">Plan</dt>
                <dd class="mt-1">
                  <span
                    :class="[
                      'px-2 py-1 text-xs font-medium rounded capitalize',
                      organization.plan === 'free' ? 'bg-gray-100 text-gray-800' :
                      organization.plan === 'pro' ? 'bg-blue-100 text-blue-800' :
                      organization.plan === 'team' ? 'bg-purple-100 text-purple-800' :
                      'bg-green-100 text-green-800'
                    ]"
                  >
                    {{ organization.plan }}
                  </span>
                </dd>
              </div>
              <div>
                <dt class="text-sm font-medium text-gray-500">Created</dt>
                <dd class="mt-1 text-sm text-gray-900">{{ formatDate(organization.created_at) }}</dd>
              </div>
            </dl>
          </div>
        </div>

        <!-- Statistics -->
        <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
          <StatCard
            title="Repositories"
            :value="stats.repo_count?.toString() || '0'"
          />
          <StatCard
            title="Reviews This Month"
            :value="stats.reviews_this_month?.toString() || '0'"
          />
          <StatCard
            title="Total Users"
            :value="users.length.toString()"
          />
        </div>

        <!-- Users List -->
        <div class="bg-white shadow rounded-lg overflow-hidden">
          <div class="px-6 py-4 border-b border-gray-200">
            <h2 class="text-lg font-semibold text-gray-900">Organization Users</h2>
          </div>
          <div class="px-6 py-4">
            <div v-if="usersLoading" class="text-center py-8 text-gray-500">
              Loading users...
            </div>
            <div v-else-if="users.length === 0" class="text-center py-8 text-gray-500">
              No users found
            </div>
            <ul v-else class="space-y-3">
              <li
                v-for="user in users"
                :key="user.id"
                class="flex items-center justify-between py-2 border-b border-gray-100 last:border-0"
              >
                <div>
                  <p class="text-sm font-medium text-gray-900">{{ user.name }}</p>
                  <p class="text-xs text-gray-500">{{ user.email }}</p>
                </div>
                <span
                  :class="[
                    'px-2 py-1 text-xs font-medium rounded',
                    user.role === 'super_admin' ? 'bg-red-100 text-red-800' :
                    user.role === 'org_admin' ? 'bg-blue-100 text-blue-800' :
                    'bg-gray-100 text-gray-800'
                  ]"
                >
                  {{ user.role }}
                </span>
              </li>
            </ul>
          </div>
        </div>

        <!-- Actions -->
        <div class="bg-white shadow rounded-lg overflow-hidden">
          <div class="px-6 py-4 border-b border-gray-200">
            <h2 class="text-lg font-semibold text-gray-900">Actions</h2>
          </div>
          <div class="px-6 py-4">
            <div class="flex gap-4">
              <button
                @click="updatePlan"
                class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                Update Plan
              </button>
              <button
                @click="viewRepositories"
                class="px-4 py-2 bg-gray-600 text-white rounded-md hover:bg-gray-700 focus:outline-none focus:ring-2 focus:ring-gray-500"
              >
                View Repositories
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import NavBar from '@/components/NavBar.vue'
import StatCard from '@/components/StatCard.vue'
import { adminAPI } from '@/api/admin'
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

interface Organization {
  id: string
  name: string
  slug: string
  plan: string
  created_at: string
  updated_at: string
}

interface User {
  id: string
  email: string
  name: string
  role: string
}

const route = useRoute()
const router = useRouter()
const loading = ref(true)
const error = ref<string | null>(null)
const organization = ref<Organization | null>(null)
const stats = ref<any>({})
const users = ref<User[]>([])
const usersLoading = ref(true)

const formatDate = (dateString: string) => {
  return new Date(dateString).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })
}

const updatePlan = () => {
  // TODO: Implement plan update
  alert('Plan update feature coming soon')
}

const viewRepositories = () => {
  // TODO: Navigate to repositories view filtered by org
  alert('Repository view feature coming soon')
}

onMounted(async () => {
  const orgId = route.params.id as string

  try {
    const [orgData, usersData] = await Promise.all([
      adminAPI.getOrganization(orgId),
      adminAPI.listUsers(),
    ])

    organization.value = orgData.organization
    stats.value = orgData.stats || {}

    // Filter users by organization
    users.value = usersData.filter((u: any) => u.org_id === orgId)
  } catch (err: any) {
    error.value = err.response?.data?.error || err.message || 'Failed to load organization details'
  } finally {
    loading.value = false
    usersLoading.value = false
  }
})
</script>
