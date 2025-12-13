<template>
  <div class="min-h-screen bg-gray-50">
    <NavBar />
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div class="mb-8">
        <h1 class="text-3xl font-bold text-gray-900">Super Admin Dashboard</h1>
        <p class="mt-2 text-gray-600">System-wide overview and management</p>
      </div>

      <div v-if="loading" class="text-center py-12">
        <p class="text-gray-500">Loading statistics...</p>
      </div>

      <div v-else-if="error" class="bg-red-50 border border-red-200 rounded-lg p-4">
        <p class="text-red-800">{{ error }}</p>
      </div>

      <div v-else>
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
          <StatCard
            title="Total Organizations"
            :value="stats.total_organizations.toString()"
          />
          <StatCard
            title="Total Users"
            :value="stats.total_users.toString()"
          />
          <StatCard
            title="Total Repositories"
            :value="stats.total_repositories.toString()"
          />
          <StatCard
            title="Reviews This Month"
            :value="stats.reviews_this_month.toString()"
          />
        </div>

        <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
          <div class="bg-white shadow rounded-lg overflow-hidden">
            <div class="px-6 py-4 border-b border-gray-200">
              <h2 class="text-lg font-semibold text-gray-900">Organizations</h2>
            </div>
            <div class="px-6 py-4">
              <div v-if="organizationsLoading" class="text-center py-8 text-gray-500">
                Loading...
              </div>
              <div v-else-if="organizations.length === 0" class="text-center py-8 text-gray-500">
                No organizations
              </div>
              <ul v-else class="space-y-3">
                <li
                  v-for="org in organizations"
                  :key="org.id"
                  class="flex items-center justify-between py-2 border-b border-gray-100 last:border-0"
                >
                  <div>
                    <p class="text-sm font-medium text-gray-900">{{ org.name }}</p>
                    <p class="text-xs text-gray-500">{{ org.slug }}</p>
                  </div>
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
                </li>
              </ul>
            </div>
          </div>

          <div class="bg-white shadow rounded-lg overflow-hidden">
            <div class="px-6 py-4 border-b border-gray-200">
              <h2 class="text-lg font-semibold text-gray-900">Recent Users</h2>
            </div>
            <div class="px-6 py-4">
              <div v-if="usersLoading" class="text-center py-8 text-gray-500">
                Loading...
              </div>
              <div v-else-if="users.length === 0" class="text-center py-8 text-gray-500">
                No users
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
                      'bg-blue-100 text-blue-800'
                    ]"
                  >
                    {{ user.role }}
                  </span>
                </li>
              </ul>
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

interface SystemStats {
  total_organizations: number
  total_users: number
  total_repositories: number
  total_reviews: number
  reviews_this_month: number
}

interface Organization {
  id: string
  name: string
  slug: string
  plan: string
}

interface User {
  id: string
  email: string
  name: string
  role: string
}

const loading = ref(true)
const error = ref<string | null>(null)
const stats = ref<SystemStats>({
  total_organizations: 0,
  total_users: 0,
  total_repositories: 0,
  total_reviews: 0,
  reviews_this_month: 0,
})

const organizations = ref<Organization[]>([])
const organizationsLoading = ref(true)
const users = ref<User[]>([])
const usersLoading = ref(true)

onMounted(async () => {
  try {
    const [statsData, orgsData, usersData] = await Promise.all([
      adminAPI.getStats(),
      adminAPI.listOrganizations(),
      adminAPI.listUsers(),
    ])

    stats.value = statsData
    organizations.value = orgsData.slice(0, 10) // Show first 10
    users.value = usersData.slice(0, 10) // Show first 10
  } catch (err: any) {
    error.value = err.response?.data?.error || err.message || 'Failed to load dashboard data'
  } finally {
    loading.value = false
    organizationsLoading.value = false
    usersLoading.value = false
  }
})
</script>
