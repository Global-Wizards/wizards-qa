<template>
  <div>
    <h2 class="text-3xl font-bold text-gray-900 mb-6">Dashboard</h2>
    
    <!-- Stats Grid -->
    <div class="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4 mb-8">
      <div class="stat-card">
        <div class="text-sm font-medium text-gray-500">Total Tests</div>
        <div class="mt-1 text-3xl font-semibold text-gray-900">{{ stats.totalTests }}</div>
      </div>
      
      <div class="stat-card">
        <div class="text-sm font-medium text-gray-500">Passed</div>
        <div class="mt-1 text-3xl font-semibold text-green-600">{{ stats.passedTests }}</div>
      </div>
      
      <div class="stat-card">
        <div class="text-sm font-medium text-gray-500">Failed</div>
        <div class="mt-1 text-3xl font-semibold text-red-600">{{ stats.failedTests }}</div>
      </div>
      
      <div class="stat-card">
        <div class="text-sm font-medium text-gray-500">Success Rate</div>
        <div class="mt-1 text-3xl font-semibold text-indigo-600">{{ stats.avgSuccessRate }}%</div>
      </div>
    </div>

    <!-- Recent Tests -->
    <div class="bg-white shadow rounded-lg p-6">
      <h3 class="text-lg font-medium text-gray-900 mb-4">Recent Tests</h3>
      <div class="space-y-3">
        <div v-for="test in stats.recentTests" :key="test.name" class="flex items-center justify-between p-3 bg-gray-50 rounded">
          <div>
            <div class="font-medium">{{ test.name }}</div>
            <div class="text-sm text-gray-500">{{ new Date(test.timestamp).toLocaleString() }}</div>
          </div>
          <span :class="['px-3 py-1 rounded-full text-sm font-medium', test.status === 'passed' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800']">
            {{ test.status }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

const stats = ref({
  totalTests: 0,
  passedTests: 0,
  failedTests: 0,
  avgSuccessRate: 0,
  recentTests: []
})

onMounted(async () => {
  const { data } = await axios.get('/api/stats')
  stats.value = data
})
</script>

<style scoped>
.stat-card {
  @apply bg-white overflow-hidden shadow rounded-lg px-4 py-5 sm:p-6;
}
</style>
