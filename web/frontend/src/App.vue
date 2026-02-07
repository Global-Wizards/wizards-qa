<template>
  <div class="min-h-screen bg-gray-50">
    <!-- Navigation -->
    <nav class="bg-white shadow-sm border-b">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex justify-between h-16">
          <div class="flex">
            <div class="flex-shrink-0 flex items-center">
              <h1 class="text-2xl font-bold text-indigo-600">🧙‍♂️ Wizards QA</h1>
            </div>
            <div class="hidden sm:ml-8 sm:flex sm:space-x-8">
              <router-link to="/" class="nav-link">Dashboard</router-link>
              <router-link to="/tests" class="nav-link">Tests</router-link>
              <router-link to="/reports" class="nav-link">Reports</router-link>
              <router-link to="/flows" class="nav-link">Flows</router-link>
            </div>
          </div>
        </div>
      </div>
    </nav>

    <!-- Main Content -->
    <main class="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
      <router-view />
    </main>
  </div>
</template>

<script setup>
// App component
</script>

<style>
.nav-link {
  @apply border-b-2 border-transparent inline-flex items-center px-1 pt-1 text-sm font-medium text-gray-900 hover:border-indigo-500 hover:text-indigo-600;
}

.nav-link.router-link-active {
  @apply border-indigo-500 text-indigo-600;
}
</style>
