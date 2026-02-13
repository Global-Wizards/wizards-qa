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
              <Badge v-if="d.status === 'completed'" variant="secondary">{{ d.flowCount }} flow(s)</Badge>
              <Badge v-else variant="destructive" class="text-[10px]">Failed</Badge>
            </div>
            <p v-if="d.error" class="text-[10px] text-destructive mt-1 truncate" :title="d.error">{{ d.error }}</p>
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- Stats Grid -->
    <div class="grid gap-4 grid-cols-2 lg:grid-cols-4">
      <StatCard title="Mechanics" :value="analysis?.mechanics?.length || 0" :icon="Cog" description="Game mechanics found" />
      <StatCard title="UI Elements" :value="analysis?.uiElements?.length || 0" :icon="Layout" description="Interface elements" />
      <StatCard title="User Flows" :value="analysis?.userFlows?.length || 0" :icon="GitBranch" description="Interaction paths" />
      <StatCard title="Edge Cases" :value="analysis?.edgeCases?.length || 0" :icon="AlertTriangle" description="Potential issues" />
      <StatCard title="UI/UX Issues" :value="analysis?.uiuxAnalysis?.length || 0" :icon="Eye" description="Visual findings" />
      <StatCard title="Wording Issues" :value="analysis?.wordingCheck?.length || 0" :icon="Type" description="Text findings" />
      <StatCard title="Game Design" :value="analysis?.gameDesign?.length || 0" :icon="Gamepad2" description="Design observations" />
      <StatCard title="Test Flows" :value="flows?.length || 0" :icon="PlayCircle" description="Generated flows" />
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

    <!-- Screenshot -->
    <Card v-if="pageMeta?.screenshotB64">
      <CardHeader>
        <CardTitle class="text-base">Screenshot</CardTitle>
      </CardHeader>
      <CardContent>
        <img
          :src="'data:image/jpeg;base64,' + pageMeta.screenshotB64"
          class="rounded-md border max-w-lg"
          alt="Game screenshot"
        />
      </CardContent>
    </Card>
  </div>
</template>

<script setup>
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import StatCard from '@/components/StatCard.vue'
import { Cog, Layout, GitBranch, AlertTriangle, Eye, Type, Gamepad2, PlayCircle } from 'lucide-vue-next'

defineProps({
  analysis: { type: Object, default: null },
  pageMeta: { type: Object, default: null },
  flows: { type: Array, default: () => [] },
  devices: { type: Array, default: () => [] },
})
</script>
