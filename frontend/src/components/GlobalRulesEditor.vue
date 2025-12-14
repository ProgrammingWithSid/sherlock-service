<template>
  <div class="bg-white shadow rounded-lg p-6">
    <div class="mb-4">
      <h3 class="text-lg font-semibold text-gray-900">{{ title }}</h3>
      <p class="mt-1 text-sm text-gray-600">{{ description }}</p>
    </div>

    <div v-if="loading" class="text-center py-4">
      <p class="text-gray-500">Loading rules...</p>
    </div>

    <div v-else>
      <!-- Rules list -->
      <div class="space-y-2 mb-4">
        <div
          v-for="(rule, index) in localRules"
          :key="index"
          class="flex items-start space-x-2 p-3 bg-gray-50 rounded-lg"
        >
          <div class="flex-1">
            <textarea
              v-model="localRules[index]"
              class="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              rows="2"
              :placeholder="`Rule ${index + 1}`"
            />
          </div>
          <button
            @click="removeRule(index)"
            class="text-red-600 hover:text-red-800 p-2"
            :disabled="localRules.length <= 1"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
        </div>
      </div>

      <!-- Add rule button -->
      <button
        @click="addRule"
        class="w-full py-2 px-4 border-2 border-dashed border-gray-300 rounded-lg text-gray-600 hover:border-blue-500 hover:text-blue-600 transition-colors"
      >
        + Add Rule
      </button>

      <!-- Info about org rules -->
      <div v-if="showOrgInfo && orgRules.length > 0" class="mt-4 p-3 bg-blue-50 border border-blue-200 rounded-lg">
        <p class="text-sm text-blue-800">
          <strong>Organization Rules:</strong> These rules are inherited from your organization settings.
          {{ localRules.length > 0 ? 'You can override them with repository-specific rules.' : 'Add rules below to override them for this repository.' }}
        </p>
        <div class="mt-2 space-y-1">
          <div
            v-for="(rule, index) in orgRules"
            :key="index"
            class="text-xs text-blue-700 bg-blue-100 px-2 py-1 rounded"
          >
            {{ rule }}
          </div>
        </div>
      </div>

      <!-- Actions -->
      <div class="mt-6 flex justify-end space-x-3">
        <button
          @click="cancel"
          class="px-4 py-2 border border-gray-300 rounded-lg text-gray-700 hover:bg-gray-50"
        >
          Cancel
        </button>
        <button
          @click="save"
          :disabled="saving"
          class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
        >
          {{ saving ? 'Saving...' : 'Save Rules' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'

interface Props {
  rules: string[]
  orgRules?: string[]
  title?: string
  description?: string
  showOrgInfo?: boolean
  loading?: boolean
}

interface Emits {
  (e: 'save', rules: string[]): void
  (e: 'cancel'): void
}

const props = withDefaults(defineProps<Props>(), {
  rules: () => [],
  orgRules: () => [],
  title: 'Global Rules',
  description: 'Configure rules that will be applied to all code reviews. These rules are passed to code-sherlock.',
  showOrgInfo: false,
  loading: false,
})

const emit = defineEmits<Emits>()

const localRules = ref<string[]>([])
const saving = ref(false)

watch(() => props.rules, (newRules) => {
  localRules.value = newRules.length > 0 ? [...newRules] : ['']
}, { immediate: true })

const addRule = () => {
  localRules.value.push('')
}

const removeRule = (index: number) => {
  if (localRules.value.length > 1) {
    localRules.value.splice(index, 1)
  }
}

const save = async () => {
  // Filter out empty rules
  const validRules = localRules.value.filter(rule => rule.trim() !== '')
  saving.value = true
  try {
    emit('save', validRules)
  } finally {
    saving.value = false
  }
}

const cancel = () => {
  // Reset to original rules
  localRules.value = props.rules.length > 0 ? [...props.rules] : ['']
  emit('cancel')
}
</script>

