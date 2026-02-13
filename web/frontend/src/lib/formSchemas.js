import { z } from 'zod'

export const loginSchema = z.object({
  email: z.string().min(1, 'Email is required').email('Invalid email address'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
})

export const registerSchema = z.object({
  displayName: z.string().min(1, 'Display name is required'),
  email: z.string().min(1, 'Email is required').email('Invalid email address'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
})

export const projectSchema = z.object({
  name: z.string().min(1, 'Project name is required'),
  gameUrl: z.string().url('Invalid URL').or(z.literal('')).optional(),
  description: z.string().optional(),
})

export const testPlanDetailsSchema = z.object({
  name: z.string().min(1, 'Plan name is required'),
  gameUrl: z.string().min(1, 'Game URL is required').url('Invalid URL'),
  description: z.string().optional(),
})
