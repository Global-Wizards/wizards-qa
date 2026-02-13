<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { useAuth } from '@/composables/useAuth'
import { loginSchema, registerSchema } from '@/lib/formSchemas'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { FormField, FormItem, FormLabel, FormControl, FormMessage } from '@/components/ui/form'
import { AnimatedGradientText } from '@/components/ui/animated-gradient-text'

const router = useRouter()
const { login, register } = useAuth()

const mode = ref('login')
const serverError = ref('')
const submitting = ref(false)

const schema = computed(() =>
  mode.value === 'register' ? registerSchema : loginSchema
)

const { handleSubmit, resetForm, meta } = useForm({
  validationSchema: computed(() => toTypedSchema(schema.value)),
})

const onSubmit = handleSubmit(async (values) => {
  serverError.value = ''
  submitting.value = true

  try {
    if (mode.value === 'register') {
      await register(values.email, values.password, values.displayName)
    } else {
      await login(values.email, values.password)
    }
    router.push('/')
  } catch (err) {
    serverError.value = err.message || 'An error occurred'
  } finally {
    submitting.value = false
  }
})

function toggleMode() {
  mode.value = mode.value === 'login' ? 'register' : 'login'
  serverError.value = ''
  resetForm()
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-background px-4">
    <Card class="w-full max-w-md">
      <CardHeader class="text-center">
        <div class="mb-2">
          <AnimatedGradientText class="text-2xl font-bold">Wizards QA</AnimatedGradientText>
        </div>
        <CardTitle>{{ mode === 'login' ? 'Sign In' : 'Create Account' }}</CardTitle>
        <CardDescription>
          {{ mode === 'login' ? 'Enter your credentials to continue' : 'Create your account to get started' }}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form @submit.prevent="onSubmit" class="space-y-4">
          <Alert v-if="serverError" variant="destructive">
            <AlertDescription>{{ serverError }}</AlertDescription>
          </Alert>

          <FormField v-if="mode === 'register'" name="displayName" v-slot="{ componentField }">
            <FormItem>
              <FormLabel>Display Name</FormLabel>
              <FormControl>
                <Input
                  v-bind="componentField"
                  type="text"
                  placeholder="Your name"
                  :disabled="submitting"
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>

          <FormField name="email" v-slot="{ componentField }">
            <FormItem>
              <FormLabel>Email</FormLabel>
              <FormControl>
                <Input
                  v-bind="componentField"
                  type="email"
                  placeholder="you@example.com"
                  :disabled="submitting"
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>

          <FormField name="password" v-slot="{ componentField }">
            <FormItem>
              <FormLabel>Password</FormLabel>
              <FormControl>
                <Input
                  v-bind="componentField"
                  type="password"
                  placeholder="Min. 8 characters"
                  :disabled="submitting"
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          </FormField>

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
