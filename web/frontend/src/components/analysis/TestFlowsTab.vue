<template>
  <div class="space-y-4">
    <div v-if="flows?.length" class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <Card
        v-for="flow in flows"
        :key="flow.name"
        class="cursor-pointer hover:shadow-md hover:border-primary/20 transition-all duration-200"
        @click="previewFlow(flow)"
      >
        <CardHeader class="pb-2">
          <CardTitle class="text-sm">{{ flow.name }}</CardTitle>
        </CardHeader>
        <CardContent>
          <div class="flex flex-wrap gap-1 mb-2">
            <Badge v-for="tag in flow.tags" :key="tag" variant="secondary" class="text-xs">{{ tag }}</Badge>
          </div>
          <p class="text-xs text-muted-foreground">{{ flow.commands?.length || 0 }} commands</p>
        </CardContent>
      </Card>
    </div>

    <!-- Empty state -->
    <div v-else class="text-center py-12 text-muted-foreground">
      <p>No test flows generated.</p>
    </div>

    <!-- YAML Preview Dialog -->
    <Dialog :open="dialogOpen" @update:open="dialogOpen = $event">
      <DialogContent class="max-w-3xl max-h-[80vh] overflow-auto">
        <DialogHeader>
          <DialogTitle>{{ selectedFlow?.name }}</DialogTitle>
          <DialogDescription>Generated flow YAML</DialogDescription>
        </DialogHeader>
        <div class="mt-4 relative">
          <Button
            variant="outline"
            size="sm"
            class="absolute top-2 right-2 z-10"
            @click="copyYaml"
          >
            {{ copied ? 'Copied!' : 'Copy' }}
          </Button>
          <pre class="bg-muted rounded-md p-4 text-sm overflow-auto max-h-[60vh]"><code>{{ yamlContent }}</code></pre>
        </div>
      </DialogContent>
    </Dialog>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useClipboard } from '@vueuse/core'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription } from '@/components/ui/dialog'

defineProps({
  flows: { type: Array, default: () => [] },
  gameUrl: { type: String, default: '' },
})

const dialogOpen = ref(false)
const selectedFlow = ref(null)
const yamlContent = ref('')
const { copied, copy } = useClipboard({ copiedDuring: 2000 })

function buildYaml(flow) {
  let yaml = ''
  if (flow.url) yaml += `url: ${flow.url}\n`
  if (flow.appId) yaml += `appId: ${flow.appId}\n`
  if (flow.tags?.length) {
    yaml += 'tags:\n'
    flow.tags.forEach((t) => { yaml += `  - ${t}\n` })
  }
  yaml += '---\n'
  if (flow.commands?.length) {
    flow.commands.forEach((cmd) => {
      for (const [key, val] of Object.entries(cmd)) {
        if (key === 'comment') {
          yaml += `# ${val}\n`
        } else if (typeof val === 'object' && val !== null) {
          yaml += `- ${key}:\n`
          for (const [sk, sv] of Object.entries(val)) {
            yaml += `    ${sk}: ${sv}\n`
          }
        } else {
          yaml += `- ${key}: ${val}\n`
        }
      }
    })
  }
  return yaml || 'No YAML content available'
}

function previewFlow(flow) {
  selectedFlow.value = flow
  yamlContent.value = buildYaml(flow)
  dialogOpen.value = true
}

async function copyYaml() {
  await copy(yamlContent.value)
}
</script>
