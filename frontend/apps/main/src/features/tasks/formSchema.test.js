import { describe, test, expect } from 'vitest'
import { createTaskFormSchema, createProjectFormSchema, createStatusFormSchema } from './formSchema'

const mockT = (key, params) => `${key} ${JSON.stringify(params || {})}`

describe('Task form schema', () => {
  const schema = createTaskFormSchema(mockT)

  test('minimal task gets defaults', () => {
    const out = schema.parse({ title: 'Call back' })
    expect(out).toMatchObject({ title: 'Call back', priority: 'medium', description: '' })
    expect(out.project_id).toBeNull()
    expect(out.due_date).toBeNull()
  })

  test('title is required and trimmed', () => {
    expect(() => schema.parse({ title: '   ' })).toThrow()
    expect(schema.parse({ title: '  x  ' }).title).toBe('x')
    expect(() => schema.parse({ title: 'a'.repeat(501) })).toThrow()
  })

  test('select ids become numbers, none clears', () => {
    const out = schema.parse({ title: 'x', project_id: '4', assigned_user_id: 'none', status_id: 2 })
    expect(out.project_id).toBe(4)
    expect(out.assigned_user_id).toBeNull()
    expect(out.status_id).toBe(2)
  })

  test('due date must be YYYY-MM-DD, empty clears', () => {
    expect(schema.parse({ title: 'x', due_date: '2026-09-30' }).due_date).toBe('2026-09-30')
    expect(schema.parse({ title: 'x', due_date: '' }).due_date).toBeNull()
    expect(() => schema.parse({ title: 'x', due_date: '30/09/2026' })).toThrow()
  })

  test('unknown priority rejected', () => {
    expect(() => schema.parse({ title: 'x', priority: 'critical' })).toThrow()
  })
})

describe('Project form schema', () => {
  const schema = createProjectFormSchema(mockT)

  test('name required', () => {
    expect(() => schema.parse({ name: '' })).toThrow()
    expect(schema.parse({ name: 'Onboarding' })).toMatchObject({ description: '', color: '' })
  })
})

describe('Task status form schema', () => {
  const schema = createStatusFormSchema(mockT)

  test('valid status', () => {
    expect(schema.parse({ name: 'Review', category: 'in_progress', position: '4' })).toMatchObject({
      position: 4,
      is_default: false
    })
  })

  test('category must be known', () => {
    expect(() => schema.parse({ name: 'Review', category: 'open' })).toThrow()
  })
})
