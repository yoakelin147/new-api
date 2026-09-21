/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useState } from 'react'
import { Code, Plus, Table, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { Dialog } from '@/components/dialog'
import type { RequestPathRewriteRule } from '../../types'

type RequestPathRewriteEditorDialogProps = {
  open: boolean
  value: string
  onOpenChange: (open: boolean) => void
  onSave: (value: string) => void
}

const REQUEST_PATH_OPTIONS = [
  '/v1/chat/completions',
  '/v1/responses',
  '/v1/messages',
] as const

const CONDITION_OPTIONS = [
  {
    value: 'request_path',
    label: 'Request path',
    defaultValue: '/v1/chat/completions',
  },
] as const

function parseRules(value: string): RequestPathRewriteRule[] | null {
  if (!value.trim()) return []
  try {
    const parsed: unknown = JSON.parse(value)
    if (
      !Array.isArray(parsed) ||
      !parsed.every(
        (rule) =>
          typeof rule === 'object' &&
          rule !== null &&
          typeof rule.target_path === 'string' &&
          typeof rule.conditions === 'object' &&
          rule.conditions !== null &&
          !Array.isArray(rule.conditions)
      )
    ) {
      return null
    }
    return parsed as RequestPathRewriteRule[]
  } catch {
    return null
  }
}

function isValidRule(rule: RequestPathRewriteRule): boolean {
  return (
    rule.target_path.startsWith('/') &&
    REQUEST_PATH_OPTIONS.includes(
      rule.conditions?.request_path as (typeof REQUEST_PATH_OPTIONS)[number]
    )
  )
}

export function RequestPathRewriteEditorDialog({
  open,
  value,
  onOpenChange,
  onSave,
}: RequestPathRewriteEditorDialogProps) {
  const { t } = useTranslation()
  const [mode, setMode] = useState<'visual' | 'json'>('visual')
  const [draft, setDraft] = useState(value)
  const [selectedRuleIndex, setSelectedRuleIndex] = useState(0)
  const rules = parseRules(draft)
  const effectiveRuleIndex = rules?.length
    ? Math.min(selectedRuleIndex, rules.length - 1)
    : 0
  const selectedRule = rules?.[effectiveRuleIndex]
  const valid = draft.trim() === '' || Boolean(rules?.every(isValidRule))

  const updateRules = (nextRules: RequestPathRewriteRule[]) => {
    setDraft(nextRules.length === 0 ? '' : JSON.stringify(nextRules, null, 2))
    if (nextRules.length > 0 && selectedRuleIndex >= nextRules.length) {
      setSelectedRuleIndex(nextRules.length - 1)
    }
  }

  const updateRule = (patch: Partial<RequestPathRewriteRule>) => {
    if (!rules || !selectedRule) return
    updateRules(
      rules.map((rule, index) =>
        index === effectiveRuleIndex ? { ...rule, ...patch } : rule
      )
    )
  }

  const addRule = () => {
    if (!rules) return
    updateRules([
      ...rules,
      {
        conditions: { request_path: '/v1/chat/completions' },
        target_path: '',
      },
    ])
    setSelectedRuleIndex(rules.length)
  }

  const removeRule = (index: number) => {
    if (!rules) return
    updateRules(rules.filter((_, ruleIndex) => ruleIndex !== index))
  }

  const updateConditionType = (
    currentType: string,
    nextType: string | null
  ) => {
    if (!selectedRule || nextType === null || currentType === nextType) return
    const conditions = { ...selectedRule.conditions }
    delete conditions[currentType]
    const option = CONDITION_OPTIONS.find((item) => item.value === nextType)
    conditions[nextType] = option?.defaultValue || ''
    updateRule({ conditions })
  }

  const updateConditionValue = (
    conditionType: string,
    conditionValue: string | null
  ) => {
    if (!selectedRule || conditionValue === null) return
    updateRule({
      conditions: {
        ...selectedRule.conditions,
        [conditionType]: conditionValue,
      },
    })
  }

  const addCondition = () => {
    if (!selectedRule) return
    const conditions = { ...selectedRule.conditions }
    const option = CONDITION_OPTIONS.find(
      (item) => conditions[item.value] === undefined
    )
    if (!option) return
    conditions[option.value] = option.defaultValue
    updateRule({ conditions })
  }

  const removeCondition = (conditionType: string) => {
    if (!selectedRule) return
    const conditions = { ...selectedRule.conditions }
    delete conditions[conditionType]
    updateRule({ conditions })
  }

  const fillTemplate = () => {
    updateRules([
      {
        conditions: { request_path: '/v1/messages' },
        target_path: '/apps/anthropic/v1/messages',
      },
    ])
    setSelectedRuleIndex(0)
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Request Path Rewrite')}
      description={t(
        'Rewrite the upstream path according to the effective request path after protocol conversion. The first matching rule wins.'
      )}
      contentHeight='min(70vh, 720px)'
      contentClassName='sm:max-w-5xl'
      bodyClassName='h-full'
      footer={
        <>
          <Button
            type='button'
            variant='outline'
            onClick={() => onOpenChange(false)}
          >
            {t('Cancel')}
          </Button>
          <Button
            type='button'
            disabled={!valid}
            onClick={() => {
              onSave(draft)
              onOpenChange(false)
            }}
          >
            {t('Save')}
          </Button>
        </>
      }
    >
      <Tabs
        value={mode}
        onValueChange={(nextMode) =>
          setMode(nextMode === 'json' ? 'json' : 'visual')
        }
        className='flex h-full min-h-0 flex-col'
      >
        <div className='flex items-center justify-between border-b pb-3'>
          <TabsList>
            <TabsTrigger value='visual'>
              <Table className='h-4 w-4' aria-hidden='true' />
              {t('Visual')}
            </TabsTrigger>
            <TabsTrigger value='json'>
              <Code className='h-4 w-4' aria-hidden='true' />
              {t('JSON')}
            </TabsTrigger>
          </TabsList>
          <Button
            type='button'
            variant='outline'
            size='sm'
            onClick={fillTemplate}
          >
            {t('Fill Template')}
          </Button>
        </div>

        {rules === null && (
          <Alert variant='destructive' className='mt-3'>
            <AlertDescription>
              {t('Path rewrite rules must be a valid JSON array')}
            </AlertDescription>
          </Alert>
        )}

        <TabsContent value='visual' className='mt-0 min-h-0 flex-1'>
          {rules !== null && (
            <div className='grid h-full min-h-[420px] md:grid-cols-[240px_1fr]'>
              <aside className='space-y-3 border-r py-4 pr-4'>
                <div className='flex items-center justify-between'>
                  <span className='text-sm font-semibold'>{t('Rule')}</span>
                  <Button
                    type='button'
                    variant='ghost'
                    size='icon'
                    onClick={addRule}
                  >
                    <Plus className='h-4 w-4' aria-hidden='true' />
                  </Button>
                </div>
                <div className='space-y-2'>
                  {rules.map((rule, index) => (
                    <div
                      key={index}
                      className={
                        index === effectiveRuleIndex
                          ? 'bg-muted flex items-center gap-1 rounded-md border p-1'
                          : 'flex items-center gap-1 rounded-md border p-1'
                      }
                    >
                      <Button
                        type='button'
                        variant='ghost'
                        className='h-auto min-w-0 flex-1 justify-start px-2 py-2 text-left'
                        onClick={() => setSelectedRuleIndex(index)}
                      >
                        <span className='min-w-0'>
                          <span className='block text-sm font-medium'>
                            {t('Rule')} #{index + 1}
                          </span>
                          <span className='text-muted-foreground block truncate text-xs'>
                            {rule.conditions?.request_path || '-'} →{' '}
                            {rule.target_path || '-'}
                          </span>
                        </span>
                      </Button>
                      <Button
                        type='button'
                        variant='ghost'
                        size='icon'
                        onClick={() => removeRule(index)}
                        aria-label={t('Delete rule')}
                      >
                        <Trash2 className='h-4 w-4' aria-hidden='true' />
                      </Button>
                    </div>
                  ))}
                </div>
                <Button
                  type='button'
                  variant='outline'
                  className='w-full'
                  onClick={addRule}
                >
                  <Plus className='mr-2 h-4 w-4' aria-hidden='true' />
                  {t('Add Rule')}
                </Button>
              </aside>

              <section className='min-w-0 space-y-6 py-4 pl-5'>
                {selectedRule ? (
                  <>
                    <div className='flex items-center justify-between'>
                      <span className='text-sm font-semibold'>
                        {t('Rule')} #{effectiveRuleIndex + 1}
                      </span>
                      <Button
                        type='button'
                        variant='ghost'
                        size='sm'
                        className='text-destructive hover:text-destructive'
                        onClick={() => removeRule(effectiveRuleIndex)}
                      >
                        <Trash2 className='mr-1 h-4 w-4' aria-hidden='true' />
                        {t('Delete')}
                      </Button>
                    </div>

                    <div className='space-y-3'>
                      <div className='flex items-center justify-between'>
                        <span className='text-sm font-medium'>
                          {t('Conditions')}
                        </span>
                        <Button
                          type='button'
                          variant='outline'
                          size='sm'
                          onClick={addCondition}
                          disabled={
                            Object.keys(selectedRule.conditions || {}).length >=
                            CONDITION_OPTIONS.length
                          }
                        >
                          <Plus
                            className='mr-1 h-3.5 w-3.5'
                            aria-hidden='true'
                          />
                          {t('Add Condition')}
                        </Button>
                      </div>
                      {Object.entries(selectedRule.conditions || {}).map(
                        ([conditionType, conditionValue]) => (
                          <div
                            key={conditionType}
                            className='grid grid-cols-[1fr_1fr_auto] gap-3'
                          >
                            <Select
                              value={conditionType}
                              onValueChange={(nextType) =>
                                updateConditionType(conditionType, nextType)
                              }
                            >
                              <SelectTrigger>
                                <SelectValue />
                              </SelectTrigger>
                              <SelectContent>
                                {CONDITION_OPTIONS.map((option) => (
                                  <SelectItem
                                    key={option.value}
                                    value={option.value}
                                  >
                                    {t(option.label)}
                                  </SelectItem>
                                ))}
                              </SelectContent>
                            </Select>
                            <Select
                              value={conditionValue}
                              onValueChange={(nextValue) =>
                                updateConditionValue(conditionType, nextValue)
                              }
                            >
                              <SelectTrigger>
                                <SelectValue />
                              </SelectTrigger>
                              <SelectContent>
                                {REQUEST_PATH_OPTIONS.map((path) => (
                                  <SelectItem key={path} value={path}>
                                    {path}
                                  </SelectItem>
                                ))}
                              </SelectContent>
                            </Select>
                            <Button
                              type='button'
                              variant='ghost'
                              size='icon'
                              onClick={() => removeCondition(conditionType)}
                              aria-label={t('Delete condition')}
                            >
                              <Trash2 className='h-4 w-4' aria-hidden='true' />
                            </Button>
                          </div>
                        )
                      )}
                    </div>

                    <div className='space-y-2 border-t pt-5'>
                      <label className='text-sm font-medium'>
                        {t('Upstream request path')}
                      </label>
                      <Input
                        value={selectedRule.target_path}
                        onChange={(event) =>
                          updateRule({ target_path: event.target.value })
                        }
                        placeholder='/apps/anthropic/v1/messages'
                      />
                    </div>
                  </>
                ) : (
                  <div className='text-muted-foreground flex h-full items-center justify-center text-sm'>
                    {t('Select a rule to edit.')}
                  </div>
                )}
              </section>
            </div>
          )}
        </TabsContent>

        <TabsContent value='json' className='mt-4 min-h-0 flex-1'>
          <Textarea
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            className='h-full min-h-[420px] resize-none font-mono text-xs'
            placeholder='[{"conditions":{"request_path":"/v1/messages"},"target_path":"/apps/anthropic/v1/messages"}]'
          />
        </TabsContent>
      </Tabs>
    </Dialog>
  )
}
