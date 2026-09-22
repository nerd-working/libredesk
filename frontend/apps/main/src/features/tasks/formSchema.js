import * as z from 'zod'
import { TASK_PRIORITIES, TASK_STATUS_CATEGORIES } from '@/constants/tasks.js'

const lengthMessage = (t, min, max) => ({ message: t('validation.minmax', { min, max }) })

// Select components hand back string ids, 'none' clears the field.
const optionalId = z
  .union([z.string(), z.number()])
  .nullable()
  .optional()
  .transform((v) => (v === '' || v === 'none' || v == null ? null : Number(v)))

export const createTaskFormSchema = (t) =>
  z.object({
    title: z
      .string({ required_error: t('globals.messages.required') })
      .trim()
      .min(1, lengthMessage(t, 1, 500))
      .max(500, lengthMessage(t, 1, 500)),
    description: z.string().max(50000).optional().default(''),
    project_id: optionalId,
    status_id: optionalId,
    priority: z.enum(TASK_PRIORITIES).default('medium'),
    assigned_user_id: optionalId,
    due_date: z
      .string()
      .regex(/^\d{4}-\d{2}-\d{2}$/)
      .or(z.literal(''))
      .nullable()
      .optional()
      .transform((v) => v || null)
  })

export const createProjectFormSchema = (t) =>
  z.object({
    name: z
      .string({ required_error: t('globals.messages.required') })
      .trim()
      .min(1, lengthMessage(t, 1, 140))
      .max(140, lengthMessage(t, 1, 140)),
    description: z.string().max(2000).optional().default(''),
    color: z.string().max(20).optional().default('')
  })

export const createStatusFormSchema = (t) =>
  z.object({
    name: z
      .string({ required_error: t('globals.messages.required') })
      .trim()
      .min(1, lengthMessage(t, 1, 140))
      .max(140, lengthMessage(t, 1, 140)),
    category: z.enum(TASK_STATUS_CATEGORIES, { required_error: t('globals.messages.required') }),
    color: z.string().max(20).optional().default(''),
    position: z.coerce.number().int().min(0).default(0),
    is_default: z.boolean().optional().default(false)
  })
