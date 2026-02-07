<template>
  <div>
    <h2 class="text-3xl font-bold text-gray-900 mb-6">Flow Templates</h2>
    
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <div v-for="flow in flows" :key="flow.name" class="bg-white overflow-hidden shadow rounded-lg hover:shadow-lg transition-shadow cursor-pointer" @click="viewFlow(flow.name)">
        <div class="p-5">
          <div class="flex items-center">
            <div class="flex-1 min-w-0">
              <h3 class="text-lg font-medium text-gray-900">{{ flow.name }}</h3>
              <p class="mt-1 text-sm text-gray-500">{{ flow.category }}</p>
            </div>
            <div class="flex-shrink-0">
              <svg class="h-6 w-6 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
              </svg>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

const flows = ref([])

onMounted(async () => {
  const { data } = await axios.get('/api/flows')
  flows.value = data.flows
})

function viewFlow(name) {
  console.log('View flow:', name)
}
</script>
