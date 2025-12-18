<template>
  <div class="min-h-screen bg-gray-50 flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
    <div class="max-w-md w-full space-y-8">
        <div>
          <h2 class="mt-6 text-center text-3xl font-extrabold text-gray-900">
            {{ isClaimingOrg ? 'Claim Your Organization' : 'Create your organization' }}
          </h2>
          <p class="mt-2 text-center text-sm text-gray-600">
            {{ isClaimingOrg ? 'Complete your account setup to access your organization dashboard' : 'Sign up to start using Code-Sherlock' }}
          </p>
          <div v-if="isClaimingOrg" class="mt-3 bg-blue-50 border border-blue-200 rounded-md p-3">
            <p class="text-sm text-blue-800">
              ✓ You're claiming an organization created via GitHub App installation
            </p>
          </div>
        </div>

      <form class="mt-8 space-y-6" @submit.prevent="handleSignup">
        <div v-if="error" class="bg-red-50 border border-red-200 rounded-md p-4">
          <p class="text-sm text-red-800">{{ error }}</p>
        </div>

        <div class="space-y-4">
          <div>
            <label for="name" class="block text-sm font-medium text-gray-700">Your Name</label>
            <input
              id="name"
              v-model="form.name"
              type="text"
              required
              class="mt-1 appearance-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm"
              placeholder="John Doe"
            />
          </div>

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
              minlength="8"
              class="mt-1 appearance-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm"
              placeholder="••••••••"
            />
          </div>

          <div v-if="!isClaimingOrg">
            <label for="org_name" class="block text-sm font-medium text-gray-700">Organization Name</label>
            <input
              id="org_name"
              v-model="form.orgName"
              type="text"
              required
              class="mt-1 appearance-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm"
              placeholder="Acme Inc"
            />
          </div>

          <div v-if="!isClaimingOrg">
            <label for="org_slug" class="block text-sm font-medium text-gray-700">Organization Slug (optional)</label>
            <input
              id="org_slug"
              v-model="form.orgSlug"
              type="text"
              class="mt-1 appearance-none relative block w-full px-3 py-2 border border-gray-300 placeholder-gray-500 text-gray-900 rounded-md focus:outline-none focus:ring-blue-500 focus:border-blue-500 focus:z-10 sm:text-sm"
              placeholder="acme-inc"
            />
            <p class="mt-1 text-xs text-gray-500">Leave empty to auto-generate</p>
          </div>
        </div>

        <div>
          <button
            type="submit"
            :disabled="loading"
            class="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {{ loading ? 'Creating account...' : 'Sign up' }}
          </button>
        </div>


        <div class="text-center">
          <router-link
            to="/login"
            class="text-sm text-blue-600 hover:text-blue-500"
          >
            Already have an account? Sign in
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
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const orgStore = useOrganizationStore()

const loading = ref(false)
const error = ref<string | null>(null)
const claimToken = ref<string>('')
const isClaimingOrg = ref(false)

const form = ref({
  name: '',
  email: '',
  password: '',
  orgName: '',
  orgSlug: '',
})

// Extract claim_token from URL if present
onMounted(() => {
  const token = route.query.claim_token as string
  if (token) {
    claimToken.value = token
    isClaimingOrg.value = true
  }
})

const handleSignup = async (): Promise<void> => {
  loading.value = true
  error.value = null

  try {
    // Build signup request - use claim_token if claiming, otherwise use org_name/org_slug
    const signupData: any = {
      name: form.value.name,
      email: form.value.email,
      password: form.value.password,
    }

    if (isClaimingOrg.value && claimToken.value) {
      // Claiming existing organization
      signupData.claim_token = claimToken.value
    } else {
      // Creating new organization
      signupData.org_name = form.value.orgName
      if (form.value.orgSlug) {
        signupData.org_slug = form.value.orgSlug
      }
    }

    const response = await authAPI.signup(signupData)

    // Set auth
    await authStore.setAuth(response.token, response.user.org_id)
    authStore.user = {
      org_id: response.user.org_id,
      name: response.user.name,
      plan: response.organization.plan,
    }

    // Set organization
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

    router.push('/')
  } catch (err: any) {
    error.value = err.response?.data?.error || err.message || 'Failed to create account'
  } finally {
    loading.value = false
  }
}

</script>
