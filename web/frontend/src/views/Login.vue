<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '@/composables/useAuth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'

const router = useRouter()
const { login, register } = useAuth()

const mode = ref('login') // 'login' or 'register'
const email = ref('')
const password = ref('')
const displayName = ref('')
const error = ref('')
const submitting = ref(false)

async function handleSubmit() {
  error.value = ''
  submitting.value = true

  try {
    if (mode.value === 'register') {
      await register(email.value, password.value, displayName.value)
    } else {
      await login(email.value, password.value)
    }
    router.push('/')
  } catch (err) {
    error.value = err.message || 'An error occurred'
  } finally {
    submitting.value = false
  }
}

function toggleMode() {
  mode.value = mode.value === 'login' ? 'register' : 'login'
  error.value = ''
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-background px-4">
    <Card class="w-full max-w-md">
      <CardHeader class="text-center">
        <div class="mb-2">
          <span class="text-2xl font-bold text-primary">Wizards QA</span>
        </div>
        <CardTitle>{{ mode === 'login' ? 'Sign In' : 'Create Account' }}</CardTitle>
        <CardDescription>
          {{ mode === 'login' ? 'Enter your credentials to continue' : 'Create your account to get started' }}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form @submit.prevent="handleSubmit" class="space-y-4">
          <Alert v-if="error" variant="destructive">
            <AlertDescription>{{ error }}</AlertDescription>
          </Alert>

          <div v-if="mode === 'register'" class="space-y-2">
            <label class="text-sm font-medium" for="displayName">Display Name</label>
            <Input
              id="displayName"
              v-model="displayName"
              type="text"
              placeholder="Your name"
              required
              :disabled="submitting"
            />
          </div>

          <div class="space-y-2">
            <label class="text-sm font-medium" for="email">Email</label>
            <Input
              id="email"
              v-model="email"
              type="email"
              placeholder="you@example.com"
              required
              :disabled="submitting"
            />
          </div>

          <div class="space-y-2">
            <label class="text-sm font-medium" for="password">Password</label>
            <Input
              id="password"
              v-model="password"
              type="password"
              placeholder="Min. 8 characters"
              required
              minlength="8"
              :disabled="submitting"
            />
          </div>

          <Button type="submit" class="w-full" :disabled="submitting">
            {{ submitting ? 'Please wait...' : (mode === 'login' ? 'Sign In' : 'Create Account') }}
          </Button>

          <p class="text-center text-sm text-muted-foreground">
            {{ mode === 'login' ? "Don't have an account?" : 'Already have an account?' }}
            <button type="button" class="text-primary hover:underline ml-1" @click="toggleMode">
              {{ mode === 'login' ? 'Sign up' : 'Sign in' }}
            </button>
          </p>
        </form>
      </CardContent>
    </Card>
  </div>
</template>
