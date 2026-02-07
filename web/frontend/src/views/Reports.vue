<template>
  <div>
    <h2 class="text-3xl font-bold text-gray-900 mb-6">Test Reports</h2>
    
    <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <div v-for="report in reports" :key="report.id" class="bg-white overflow-hidden shadow rounded-lg hover:shadow-lg transition-shadow cursor-pointer" @click="viewReport(report.id)">
        <div class="p-5">
          <div class="flex items-center">
            <div class="flex-1 min-w-0">
              <h3 class="text-lg font-medium text-gray-900 truncate">{{ report.name }}</h3>
              <p class="mt-1 text-sm text-gray-500">{{ report.format }}</p>
            </div>
          </div>
          <div class="mt-4 flex items-center text-sm text-gray-500">
            <span>{{ new Date(report.timestamp).toLocaleDateString() }}</span>
            <span class="ml-auto">{{ report.size }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

const reports = ref([])

onMounted(async () => {
  const { data } = await axios.get('/api/reports')
  reports.value = data.reports
})

function viewReport(id) {
  console.log('View report:', id)
}
</script>
