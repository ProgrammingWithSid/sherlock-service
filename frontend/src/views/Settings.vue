<template>
  <div class="min-h-screen bg-gray-50">
    <NavBar />
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div class="mb-8">
        <h1 class="text-3xl font-bold text-gray-900">Settings</h1>
        <p class="mt-2 text-gray-600">Manage your organization settings</p>
      </div>

      <div class="bg-white shadow rounded-lg p-6">
        <h2 class="text-xl font-semibold text-gray-900 mb-4">Organization</h2>
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
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import NavBar from '@/components/NavBar.vue'
import { useOrganizationStore } from '@/stores/organization'

const orgStore = useOrganizationStore()
</script>


