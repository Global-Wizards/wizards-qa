<template>
  <div class="space-y-6">
    <!-- Game Info Card -->
    <Card v-if="analysis?.gameInfo">
      <CardHeader>
        <CardTitle>Game Information</CardTitle>
      </CardHeader>
      <CardContent class="space-y-3">
        <div class="grid gap-4 sm:grid-cols-2">
          <div>
            <span class="text-sm text-muted-foreground">Name</span>
            <p class="font-medium">{{ analysis.gameInfo.name || 'Unknown' }}</p>
          </div>
          <div v-if="analysis.gameInfo.genre">
            <span class="text-sm text-muted-foreground">Genre</span>
            <p class="font-medium">{{ analysis.gameInfo.genre }}</p>
          </div>
          <div v-if="analysis.gameInfo.technology">
            <span class="text-sm text-muted-foreground">Technology</span>
            <p class="font-medium">{{ analysis.gameInfo.technology }}</p>
          </div>
        </div>
        <div v-if="analysis.gameInfo.description">
          <span class="text-sm text-muted-foreground">Description</span>
          <p class="text-sm mt-1">{{ analysis.gameInfo.description }}</p>
        </div>
        <div v-if="analysis.gameInfo.features?.length" class="flex flex-wrap gap-1">
          <Badge v-for="f in analysis.gameInfo.features" :key="f" variant="secondary">{{ f }}</Badge>
        </div>
      </CardContent>
    </Card>

    <!-- Device Summary -->
    <Card v-if="devices.length > 0">
      <CardHeader>
        <CardTitle class="text-base">Device Summary</CardTitle>
      </CardHeader>
      <CardContent>
        <div class="grid gap-2 sm:grid-cols-3">
          <div
            v-for="d in devices"
            :key="d.device"
            class="rounded-md border p-3 text-center"
            :class="d.status === 'failed' ? 'border-destructive/50 bg-destructive/5' : 'border-border'"
          >
            <p class="text-sm font-medium capitalize">{{ d.device === 'ios' ? 'iOS' : d.device }}</p>
            <p class="text-[10px] text-muted-foreground">{{ d.viewport }}</p>
            <div class="mt-1">
              <Badge v-if="d.status === 'completed'" variant="secondary">{{ d.flowCount }} {{ d.flowCount === 1 ? 'flow' : 'flows' }}</Badge>
              <Badge v-else variant="destructive" class="text-[10px]">Failed</Badge>
            </div>
            <p v-if="d.error" class="text-[10px] text-destructive mt-1 truncate" :title="d.error">{{ d.error }}</p>
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- Stats Grid -->
    <div v-if="visibleStats.length" class="grid gap-4 grid-cols-2 lg:grid-cols-4">
      <StatCard v-for="s in visibleStats" :key="s.title" :title="s.title" :value="s.value" :icon="s.icon" :description="s.description" />
    </div>

    <!-- Page Metadata Card -->
    <Card v-if="pageMeta">
      <CardHeader>
        <CardTitle class="text-base">Page Metadata</CardTitle>
      </CardHeader>
      <CardContent class="space-y-2 text-sm">
        <div class="grid gap-3 sm:grid-cols-2">
          <div>
            <span class="text-muted-foreground">Framework</span>
            <p class="font-medium capitalize">{{ pageMeta.framework || 'unknown' }}</p>
          </div>
          <div>
            <span class="text-muted-foreground">Canvas</span>
            <p class="font-medium">{{ pageMeta.canvasFound ? 'Detected' : 'Not found' }}</p>
          </div>
          <div v-if="pageMeta.title">
            <span class="text-muted-foreground">Page Title</span>
            <p class="font-medium">{{ pageMeta.title }}</p>
          </div>
          <div v-if="pageMeta.scriptSrcs?.length">
            <span class="text-muted-foreground">Scripts Found</span>
            <p class="font-medium">{{ pageMeta.scriptSrcs.length }}</p>
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- Screenshots -->
    <Card v-if="screenshots.length">
      <CardHeader>
        <CardTitle class="text-base">Screenshots ({{ screenshots.length }})</CardTitle>
      </CardHeader>
      <CardContent>
        <div class="grid gap-4" :class="screenshots.length > 1 ? 'sm:grid-cols-2 lg:grid-cols-3' : ''">
          <img
            v-for="(ss, i) in screenshots"
            :key="i"
            :src="'data:image/jpeg;base64,' + ss"
            class="rounded-md border w-full cursor-pointer hover:opacity-90 transition-opacity"
            :alt="'Game screenshot ' + (i + 1)"
            @click="expandedScreenshot = ss"
          />
        </div>
      </CardContent>
    </Card>

    <!-- Expanded screenshot dialog -->
    <Dialog :open="!!expandedScreenshot" @update:open="expandedScreenshot = null">
      <DialogContent class="max-w-4xl p-2">
        <img
          v-if="expandedScreenshot"
          :src="'data:image/jpeg;base64,' + expandedScreenshot"
          class="w-full rounded-md"
          alt="Game screenshot (expanded)"
        />
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import { Badge } from '@/components/ui/badge'
import StatCard from '@/components/StatCard.vue'
import { Cog, Layout, GitBranch, AlertTriangle, Eye, Type, Gamepad2, PlayCircle } from 'lucide-vue-next'

const props = defineProps({
  analysis: { type: Object, default: null },
  pageMeta: { type: Object, default: null },
  flows: { type: Array, default: () => [] },
  devices: { type: Array, default: () => [] },
})

const expandedScreenshot = ref(null)

const screenshots = computed(() => {
  if (props.pageMeta?.screenshots?.length) return props.pageMeta.screenshots
  if (props.pageMeta?.screenshotB64) return [props.pageMeta.screenshotB64]
  return []
})

const visibleStats = computed(() => {
  const all = [
    { title: 'Mechanics', value: props.analysis?.mechanics?.length || 0, icon: Cog, description: 'Game mechanics found' },
    { title: 'UI Elements', value: props.analysis?.uiElements?.length || 0, icon: Layout, description: 'Interface elements' },
    { title: 'User Flows', value: props.analysis?.userFlows?.length || 0, icon: GitBranch, description: 'Interaction paths' },
    { title: 'Edge Cases', value: props.analysis?.edgeCases?.length || 0, icon: AlertTriangle, description: 'Potential issues' },
    { title: 'UI/UX Issues', value: props.analysis?.uiuxAnalysis?.length || 0, icon: Eye, description: 'Visual findings' },
    { title: 'Wording Issues', value: props.analysis?.wordingCheck?.length || 0, icon: Type, description: 'Text findings' },
    { title: 'Game Design', value: props.analysis?.gameDesign?.length || 0, icon: Gamepad2, description: 'Design observations' },
    { title: 'Test Flows', value: props.flows?.length || 0, icon: PlayCircle, description: 'Generated flows' },
  ]
  return all.filter(s => s.value > 0)
})
</script>
