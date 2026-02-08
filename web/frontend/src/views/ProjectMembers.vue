<template>
  <div>
    <div class="flex items-center justify-between mb-6">
      <div>
        <h2 class="text-3xl font-bold tracking-tight">Members</h2>
        <p class="text-muted-foreground">Manage team members for {{ currentProject?.name || 'this project' }}</p>
      </div>
    </div>

    <div class="max-w-2xl space-y-6">
      <!-- Add Member -->
      <Card>
        <CardHeader>
          <CardTitle class="text-lg">Add Member</CardTitle>
        </CardHeader>
        <CardContent>
          <div class="flex gap-2">
            <Input v-model="newEmail" placeholder="Enter email address" class="flex-1" @keyup.enter="addMember" />
            <Select v-model="newRole">
              <SelectTrigger class="w-[130px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="member">Member</SelectItem>
                <SelectItem value="admin">Admin</SelectItem>
              </SelectContent>
            </Select>
            <Button @click="addMember" :disabled="!newEmail.trim() || adding">
              {{ adding ? 'Adding...' : 'Add' }}
            </Button>
          </div>
          <p v-if="addError" class="text-sm text-destructive mt-2">{{ addError }}</p>
          <p v-if="removeError" class="text-sm text-destructive mt-2">{{ removeError }}</p>
        </CardContent>
      </Card>

      <!-- Members List -->
      <Card>
        <CardHeader>
          <CardTitle class="text-lg">Team Members</CardTitle>
        </CardHeader>
        <CardContent>
          <template v-if="loading">
            <div class="space-y-3">
              <div v-for="i in 3" :key="i" class="h-12 rounded-md bg-muted animate-pulse" />
            </div>
          </template>
          <template v-else>
            <div class="space-y-3">
              <div
                v-for="member in members"
                :key="member.id"
                class="flex items-center justify-between p-3 rounded-md border"
              >
                <div class="flex items-center gap-3">
                  <div class="h-8 w-8 rounded-full bg-primary/10 text-primary flex items-center justify-center text-sm font-bold">
                    {{ (member.displayName || member.email).charAt(0).toUpperCase() }}
                  </div>
                  <div>
                    <p class="text-sm font-medium">{{ member.displayName || member.email }}</p>
                    <p class="text-xs text-muted-foreground">{{ member.email }}</p>
                  </div>
                </div>
                <div class="flex items-center gap-2">
                  <Badge variant="secondary">{{ member.role }}</Badge>
                  <Button
                    v-if="member.role !== 'owner'"
                    variant="ghost"
                    size="sm"
                    @click="removeMember(member)"
                  >
                    <Trash2 class="h-3 w-3 text-destructive" />
                  </Button>
                </div>
              </div>
            </div>
            <p v-if="!members.length" class="text-center text-muted-foreground py-8">
              No members yet. Add team members by email.
            </p>
          </template>
        </CardContent>
      </Card>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { Trash2 } from 'lucide-vue-next'
import { projectsApi } from '@/lib/api'
import { useProject } from '@/composables/useProject'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'

const route = useRoute()
const { currentProject } = useProject()
const loading = ref(true)
const members = ref([])
const newEmail = ref('')
const newRole = ref('member')
const adding = ref(false)
const addError = ref(null)
const removeError = ref(null)

const projectId = () => route.params.projectId

async function loadMembers() {
  try {
    const data = await projectsApi.members.list(projectId())
    members.value = data.members || []
  } catch (err) {
    console.error('Failed to load members:', err)
  } finally {
    loading.value = false
  }
}

async function addMember() {
  if (!newEmail.value.trim()) return
  adding.value = true
  addError.value = null
  try {
    const member = await projectsApi.members.add(projectId(), {
      email: newEmail.value.trim(),
      role: newRole.value,
    })
    members.value.push(member)
    newEmail.value = ''
  } catch (err) {
    addError.value = err.message
  } finally {
    adding.value = false
  }
}

async function removeMember(member) {
  removeError.value = null
  try {
    await projectsApi.members.remove(projectId(), member.userId)
    members.value = members.value.filter((m) => m.id !== member.id)
  } catch (err) {
    removeError.value = 'Failed to remove member: ' + err.message
  }
}

onMounted(loadMembers)
</script>
