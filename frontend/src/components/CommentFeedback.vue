<template>
  <div class="mt-3 flex items-center gap-2">
    <span class="text-xs text-gray-500">Was this helpful?</span>
    <button
      @click="submitFeedback('accepted')"
      :disabled="submitting || feedbackGiven"
      :class="[
        'px-2 py-1 text-xs rounded transition-colors',
        feedbackGiven === 'accepted'
          ? 'bg-green-100 text-green-700 border border-green-300'
          : 'bg-gray-100 text-gray-700 hover:bg-green-50 border border-gray-300',
        (submitting || feedbackGiven) && feedbackGiven !== 'accepted' ? 'opacity-50 cursor-not-allowed' : ''
      ]"
      title="Mark as helpful"
    >
      ✓ Accepted
    </button>
    <button
      @click="submitFeedback('dismissed')"
      :disabled="submitting || feedbackGiven"
      :class="[
        'px-2 py-1 text-xs rounded transition-colors',
        feedbackGiven === 'dismissed'
          ? 'bg-red-100 text-red-700 border border-red-300'
          : 'bg-gray-100 text-gray-700 hover:bg-red-50 border border-gray-300',
        (submitting || feedbackGiven) && feedbackGiven !== 'dismissed' ? 'opacity-50 cursor-not-allowed' : ''
      ]"
      title="Dismiss this comment"
    >
      ✗ Dismissed
    </button>
    <button
      @click="submitFeedback('fixed')"
      :disabled="submitting || feedbackGiven"
      :class="[
        'px-2 py-1 text-xs rounded transition-colors',
        feedbackGiven === 'fixed'
          ? 'bg-blue-100 text-blue-700 border border-blue-300'
          : 'bg-gray-100 text-gray-700 hover:bg-blue-50 border border-gray-300',
        (submitting || feedbackGiven) && feedbackGiven !== 'fixed' ? 'opacity-50 cursor-not-allowed' : ''
      ]"
      title="Mark as fixed"
    >
      ✓ Fixed
    </button>
    <span v-if="submitting" class="text-xs text-gray-500">Saving...</span>
    <span v-else-if="feedbackGiven" class="text-xs text-green-600">✓ Saved</span>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { feedbackAPI } from '@/api/feedback';

const props = defineProps<{
  reviewId: string;
  commentId: string;
  filePath: string;
  lineNumber: number;
}>();

const submitting = ref(false);
const feedbackGiven = ref<'accepted' | 'dismissed' | 'fixed' | null>(null);

const submitFeedback = async (feedback: 'accepted' | 'dismissed' | 'fixed') => {
  if (submitting.value || feedbackGiven.value) return;

  submitting.value = true;
  try {
    await feedbackAPI.record({
      review_id: props.reviewId,
      comment_id: props.commentId,
      file_path: props.filePath,
      line_number: props.lineNumber,
      feedback,
    });
    feedbackGiven.value = feedback;
  } catch (error) {
    console.error('Failed to submit feedback:', error);
    alert('Failed to submit feedback. Please try again.');
  } finally {
    submitting.value = false;
  }
};
</script>

