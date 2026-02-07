<template>
  <div>
    <h2 class="text-3xl font-bold text-gray-900 mb-6">Test History</h2>
    
    <div class="bg-white shadow rounded-lg overflow-hidden">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Test Name</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Status</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Duration</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Success Rate</th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Timestamp</th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-for="test in tests" :key="test.id" class="hover:bg-gray-50 cursor-pointer" @click="viewTest(test.id)">
            <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{{ test.name }}</td>
            <td class="px-6 py-4 whitespace-nowrap">
              <span :class="['px-2 py-1 inline-flex text-xs leading-5 font-semibold rounded-full', test.status === 'passed' ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800']">
                {{ test.status }}
              </span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ test.duration }}</td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ test.successRate }}%</td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{{ new Date(test.timestamp).toLocaleString() }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

const tests = ref([])

onMounted(async () => {
  const { data } = await axios.get('/api/tests')
  tests.value = data.tests
})

function viewTest(id) {
  console.log('View test:', id)
}
</script>
