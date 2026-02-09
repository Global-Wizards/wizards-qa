<template>
  <div class="space-y-6">
    <!-- Mechanics -->
    <Card v-if="analysis?.mechanics?.length">
      <CardHeader>
        <CardTitle class="text-base">Mechanics ({{ analysis.mechanics.length }})</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Description</TableHead>
              <TableHead>Actions</TableHead>
              <TableHead>Expected</TableHead>
              <TableHead>Priority</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="m in analysis.mechanics" :key="m.name">
              <TableCell class="font-medium">{{ m.name }}</TableCell>
              <TableCell class="text-sm">{{ m.description }}</TableCell>
              <TableCell>
                <div class="flex flex-wrap gap-1">
                  <Badge v-for="a in m.actions" :key="a" variant="secondary" class="text-xs">{{ a }}</Badge>
                </div>
              </TableCell>
              <TableCell class="text-sm">{{ m.expectedBehavior }}</TableCell>
              <TableCell>
                <Badge v-if="m.priority" variant="outline">{{ m.priority }}</Badge>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </CardContent>
    </Card>

    <!-- UI Elements -->
    <Card v-if="analysis?.uiElements?.length">
      <CardHeader>
        <CardTitle class="text-base">UI Elements ({{ analysis.uiElements.length }})</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Type</TableHead>
              <TableHead>Selector</TableHead>
              <TableHead>Location</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="el in analysis.uiElements" :key="el.name">
              <TableCell class="font-medium">{{ el.name }}</TableCell>
              <TableCell><Badge variant="secondary">{{ el.type }}</Badge></TableCell>
              <TableCell class="font-mono text-xs">{{ el.selector }}</TableCell>
              <TableCell class="text-sm">{{ el.location }}</TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </CardContent>
    </Card>

    <!-- User Flows -->
    <Card v-if="analysis?.userFlows?.length">
      <CardHeader>
        <CardTitle class="text-base">User Flows ({{ analysis.userFlows.length }})</CardTitle>
      </CardHeader>
      <CardContent>
        <div class="grid gap-4 md:grid-cols-2">
          <div v-for="f in analysis.userFlows" :key="f.name" class="rounded-md border p-4 space-y-2">
            <div class="flex items-center justify-between">
              <span class="font-medium text-sm">{{ f.name }}</span>
              <Badge v-if="f.priority" variant="outline">{{ f.priority }}</Badge>
            </div>
            <p class="text-sm text-muted-foreground">{{ f.description }}</p>
            <ol v-if="f.steps?.length" class="list-decimal list-inside text-sm space-y-0.5">
              <li v-for="(step, i) in f.steps" :key="i">{{ step }}</li>
            </ol>
            <p v-if="f.expectedOutcome" class="text-xs text-muted-foreground">
              <span class="font-medium">Expected:</span> {{ f.expectedOutcome }}
            </p>
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- Edge Cases -->
    <Card v-if="analysis?.edgeCases?.length">
      <CardHeader>
        <CardTitle class="text-base">Edge Cases ({{ analysis.edgeCases.length }})</CardTitle>
      </CardHeader>
      <CardContent>
        <div class="grid gap-4 md:grid-cols-2">
          <div v-for="ec in analysis.edgeCases" :key="ec.name" class="rounded-md border p-4 space-y-2">
            <span class="font-medium text-sm">{{ ec.name }}</span>
            <p class="text-sm text-muted-foreground">{{ ec.description }}</p>
            <p v-if="ec.scenario" class="text-xs">
              <span class="text-muted-foreground font-medium">Scenario:</span> {{ ec.scenario }}
            </p>
            <p v-if="ec.expectedBehavior" class="text-xs text-muted-foreground">
              <span class="font-medium">Expected:</span> {{ ec.expectedBehavior }}
            </p>
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- Empty state -->
    <div v-if="isEmpty" class="text-center py-12 text-muted-foreground">
      <p>No functional QA data available for this analysis.</p>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Table, TableHeader, TableRow, TableHead, TableBody, TableCell } from '@/components/ui/table'

const props = defineProps({
  analysis: { type: Object, default: null },
})

const isEmpty = computed(() => {
  if (!props.analysis) return true
  return !(
    props.analysis.mechanics?.length ||
    props.analysis.uiElements?.length ||
    props.analysis.userFlows?.length ||
    props.analysis.edgeCases?.length
  )
})
</script>
