<script setup>
import Sidebar from './Sidebar.vue'
</script>

<template>
  <div class="flex min-h-screen bg-background">
    <Sidebar />
    <main class="flex-1 overflow-auto">
      <div class="container py-6 px-4 md:px-8 max-w-7xl">
        <slot />
      </div>
    </main>
  </div>
</template>
