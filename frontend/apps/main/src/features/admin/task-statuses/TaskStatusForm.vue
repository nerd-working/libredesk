<template>
  <form class="space-y-4">
    <FormField v-slot="{ componentField }" name="name">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.name') }}</FormLabel>
        <FormControl>
          <Input type="text" v-bind="componentField" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="category">
      <FormItem>
        <FormLabel>{{ $t('globals.terms.category') }}</FormLabel>
        <Select v-bind="componentField">
          <FormControl>
            <SelectTrigger class="w-full"><SelectValue /></SelectTrigger>
          </FormControl>
          <SelectContent>
            <SelectItem v-for="c in TASK_STATUS_CATEGORIES" :key="c" :value="c">
              {{ $t(`tasks.category.${c}`) }}
            </SelectItem>
          </SelectContent>
        </Select>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ value, handleChange }" name="color">
      <FormItem>
        <FormLabel>{{ $t('tasks.color') }}</FormLabel>
        <FormControl>
          <ColorSwatches :model-value="value" @update:model-value="handleChange" />
        </FormControl>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ componentField }" name="position">
      <FormItem>
        <FormLabel>{{ $t('tasks.position') }}</FormLabel>
        <FormControl>
          <Input type="number" min="0" v-bind="componentField" />
        </FormControl>
        <FormDescription>{{ $t('tasks.positionHelp') }}</FormDescription>
        <FormMessage />
      </FormItem>
    </FormField>

    <FormField v-slot="{ value, handleChange }" name="is_default" type="checkbox">
      <FormItem class="flex items-center gap-2 space-y-0">
        <FormControl>
          <Checkbox :checked="value" @update:checked="handleChange" />
        </FormControl>
        <FormLabel class="font-normal">{{ $t('tasks.defaultStatusHelp') }}</FormLabel>
      </FormItem>
    </FormField>

    <slot name="footer"></slot>
  </form>
</template>

<script setup>
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@shared-ui/components/ui/form'
import { Input } from '@shared-ui/components/ui/input'
import { Checkbox } from '@shared-ui/components/ui/checkbox'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import ColorSwatches from '@/features/tasks/ColorSwatches.vue'
import { TASK_STATUS_CATEGORIES } from '@/constants/tasks.js'
</script>
