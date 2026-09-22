import { describe, test, expect } from 'vitest'
import {
  dueBucket,
  groupByDueBucket,
  isOverdue,
  localDateString,
  positionBetween,
  toTaskPayload
} from './taskUtils'

const today = '2026-09-22'
const task = (over = {}) => ({ id: 1, status_category: 'todo', due_date: null, ...over })

describe('due dates', () => {
  test('localDateString pads month and day', () => {
    expect(localDateString(new Date(2026, 0, 5))).toBe('2026-01-05')
  })

  test('overdue only when open and past due', () => {
    expect(isOverdue(task({ due_date: '2026-09-21' }), today)).toBe(true)
    expect(isOverdue(task({ due_date: today }), today)).toBe(false)
    expect(isOverdue(task({ due_date: '2026-09-21', status_category: 'done' }), today)).toBe(false)
    expect(isOverdue(task(), today)).toBe(false)
  })

  test('buckets', () => {
    expect(dueBucket(task({ due_date: '2026-09-01' }), today)).toBe('overdue')
    expect(dueBucket(task({ due_date: today }), today)).toBe('today')
    expect(dueBucket(task({ due_date: '2026-10-01' }), today)).toBe('upcoming')
    expect(dueBucket(task(), today)).toBe('noDueDate')
    expect(dueBucket(task({ due_date: '2026-09-01', status_category: 'done' }), today)).toBe('done')
  })

  test('groups keep order and skip empty ones', () => {
    const groups = groupByDueBucket(
      [task({ id: 1 }), task({ id: 2, due_date: '2026-09-01' }), task({ id: 3, due_date: today })],
      today
    )
    expect(groups.map((g) => g.key)).toEqual(['overdue', 'today', 'noDueDate'])
  })
})

describe('positionBetween', () => {
  test('empty column', () => expect(positionBetween(null, null)).toBe(1))
  test('top of column', () => expect(positionBetween(null, 3)).toBe(2))
  test('bottom of column', () => expect(positionBetween(3, null)).toBe(4))
  test('between two cards', () => expect(positionBetween(2, 3)).toBe(2.5))
})

describe('toTaskPayload', () => {
  test('keeps every field and applies overrides', () => {
    const payload = toTaskPayload(
      {
        title: 'x',
        description: null,
        project_id: 2,
        status_id: 1,
        priority: 'high',
        assigned_user_id: null,
        conversation_uuid: null,
        due_date: '2026-09-30'
      },
      { status_id: 3 }
    )
    expect(payload).toEqual({
      title: 'x',
      description: '',
      project_id: 2,
      status_id: 3,
      priority: 'high',
      assigned_user_id: null,
      conversation_uuid: '',
      due_date: '2026-09-30'
    })
  })
})
