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
import {
  createContext,
  type ReactNode,
  useContext,
  useEffect,
  useState,
} from 'react'
import { useQuery } from '@tanstack/react-query'
import * as z from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import {
  ChevronRight,
  Code2,
  Copy,
  HelpCircle,
  Palette,
  Plus,
  Trash2,
} from 'lucide-react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Button } from '@/components/ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  NativeSelect,
  NativeSelectOption,
} from '@/components/ui/native-select'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Switch } from '@/components/ui/switch'
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { MultiSelect, type Option as MultiSelectOption } from '@/components/multi-select'
import {
  sideDrawerContentClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import { cn } from '@/lib/utils'
import { getChannels } from '@/features/channels/api'
import { CHANNEL_TYPE_OPTIONS } from '@/features/channels/constants'
import type { Channel } from '@/features/channels/types'
import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

const modelEndpointTemplatesExample = JSON.stringify(
  {
    'image-generation-async': {
      path: '/v1/images/generations/async',
      method: 'POST',
      description: '提交图片生成任务，返回任务处理结果或任务信息。',
      request_example: {
        model: '{model}',
        prompt: 'A serene koi pond at sunset, ukiyo-e style.',
        size: '1024x1024',
        n: 1,
      },
      parameters: [
        {
          name: 'prompt',
          type: 'string',
          required: true,
          description: '想要生成图像的文字描述',
        },
        {
          name: 'size',
          type: 'enum',
          default: '1024x1024',
          enum_values: ['1024x1024'],
          description: '输出图像尺寸',
        },
        {
          name: 'n',
          type: 'integer',
          default: 1,
          range: '1 ~ 10',
          description: '生成图片数量',
        },
      ],
    },
    'openai-video': {
      path: '/v1/videos',
      method: 'POST',
    },
  },
  null,
  2
)

const modelEndpointRequestExample = {
  model: '{model}',
  prompt: 'A serene koi pond at sunset, ukiyo-e style.',
  size: '1024x1024',
  n: 1,
}

const modelEndpointParametersExample = [
  {
    name: 'prompt',
    type: 'string',
    required: true,
    description: '想要生成图像的文字描述',
  },
  {
    name: 'size',
    type: 'enum',
    default: '1024x1024',
    enum_values: ['1024x1024'],
    description: '输出图像尺寸',
  },
  {
    name: 'n',
    type: 'integer',
    default: 1,
    range: '1 ~ 10',
    description: '生成图片数量',
  },
]

const defaultAsyncTaskResponseMapping = {
  submit_task_id_paths: [
    'id',
    'task_id',
    'data.id',
    'data.task_id',
    'data.taskId',
    'task.id',
  ],
  submit_status_paths: ['status', 'data.status', 'task.status'],
  fetch_status_paths: [
    'status',
    'data.status',
    'data.state',
    'state',
    'task.status',
  ],
  fetch_status_map: {
    queued: 'QUEUED',
    processing: 'IN_PROGRESS',
    completed: 'SUCCESS',
    failed: 'FAILURE',
  },
  fetch_progress_paths: [
    'progress',
    'percent',
    'data.progress',
    'data.percent',
    'task.progress',
  ],
  fetch_result_url_paths: [
    'video_url',
    'image_url',
    'url',
    'data.url',
    'data.result.url',
    'data.result.images.0.url',
    'result.url',
    'result.images.0.url',
  ],
  fetch_error_message_paths: [
    'error.message',
    'error',
    'message',
    'data.error.message',
    'data.error',
    'data.message',
    'reason',
    'data.reason',
  ],
}

const asyncTaskStripFieldsExample = ['metadata.debug', 'response_format']

const asyncTaskHeadersExample = {
  Authorization: 'Bearer {api_key}',
}

const asyncTaskSetFieldsExample = {
  model: '{upstream_model}',
}

const asyncTaskParameterMappingsExample = [
  {
    name: '1:1 2K to 2048x2048',
    when: { size: '1:1', resolution: '2k' },
    strip_fields: ['resolution'],
    set_fields: { size: '2048x2048' },
  },
]

const asyncTaskBodyExample = {
  model: '{upstream_model}',
  prompt: '{body:prompt}',
  size: '{body:size}',
}

const asyncTaskSubmitResponseExample = {
  id: 'task_123456',
  status: 'queued',
}

const asyncTaskFetchResponseExample = {
  status: 'completed',
  progress: 100,
  data: {
    result: {
      images: [{ url: 'https://example.com/image.png' }],
    },
  },
}

const asyncTaskUpstreamTemplatesExample = JSON.stringify(
  [
    {
      enabled: true,
      name: '兔子 gpt-image-2 异步生图',
      channel_id: 19,
      request_path: '/v1/images/generations/async',
      model: 'gpt-image-2',
      name_rule: 0,
      submit_path: '/v1/videos',
      fetch_path: '/v1/videos/{task_id}',
      method: 'POST',
      headers: asyncTaskHeadersExample,
      strip_fields: asyncTaskStripFieldsExample,
      set_fields: asyncTaskSetFieldsExample,
      parameter_mappings: asyncTaskParameterMappingsExample,
      body: asyncTaskBodyExample,
      response_mapping: defaultAsyncTaskResponseMapping,
      submit_response_example: asyncTaskSubmitResponseExample,
      fetch_response_example: asyncTaskFetchResponseExample,
    },
    {
      enabled: true,
      name: '某供应商图片异步专用路径',
      channel_type: 1,
      request_path: '/v1/images/generations/async',
      model: 'image-2',
      name_rule: 0,
      submit_path: '/v1/images/generations/async',
      fetch_path: '/v1/images/generations/async/{task_id}',
      strip_fields: asyncTaskStripFieldsExample,
      set_fields: asyncTaskSetFieldsExample,
      body: asyncTaskBodyExample,
    },
  ],
  null,
  2
)

const supportedModelEndpointPaths = [
  {
    value: '/v1/chat/completions',
    label: 'Chat Completions',
    method: 'POST',
  },
  {
    value: '/v1/responses',
    label: 'Responses',
    method: 'POST',
  },
  {
    value: '/v1/images/generations',
    label: 'Image generations',
    method: 'POST',
  },
  {
    value: '/v1/images/generations/async',
    label: 'Async image generations',
    method: 'POST',
  },
  {
    value: '/v1/videos',
    label: 'Videos',
    method: 'POST',
  },
]

const supportedAsyncRequestPaths = supportedModelEndpointPaths.filter(
  (option) =>
    option.value === '/v1/images/generations/async' ||
    option.value === '/v1/videos'
)

const jsonObjectString = z.string().refine((value) => {
  const trimmed = value.trim()
  if (!trimmed) return true
  try {
    const parsed = JSON.parse(trimmed)
    return parsed !== null && !Array.isArray(parsed) && typeof parsed === 'object'
  } catch {
    return false
  }
}, 'Invalid JSON object format')

const jsonArrayString = z.string().refine((value) => {
  const trimmed = value.trim()
  if (!trimmed) return true
  try {
    return Array.isArray(JSON.parse(trimmed))
  } catch {
    return false
  }
}, 'Invalid JSON array format')

const schema = z.object({
  ModelEndpointTemplates: jsonObjectString,
  AsyncTaskUpstreamTemplates: jsonArrayString,
})

type EndpointTemplateSettingsFormValues = z.infer<typeof schema>

type EndpointTemplateSettingsCardProps = {
  defaultValues: EndpointTemplateSettingsFormValues
}

function normalizeJsonText(value: string, fallback: string) {
  const trimmed = (value ?? '').toString().trim()
  return trimmed ? trimmed : fallback
}

function formatJson(value: string, fallback: string) {
  const normalized = normalizeJsonText(value, fallback)
  return JSON.stringify(JSON.parse(normalized), null, 2)
}

type AsyncTaskUpstreamTemplateConfig = {
  enabled?: boolean
  name?: string
  channel_id?: number
  channel_type?: number
  request_path?: string
  model?: string
  name_rule?: number
  submit_path?: string
  fetch_path?: string
  method?: string
  headers?: Record<string, string>
  strip_fields?: string[]
  set_fields?: Record<string, unknown>
  parameter_mappings?: AsyncTaskParameterMappingConfig[]
  body?: Record<string, unknown>
  response_mapping?: AsyncTaskResponseMappingConfig
  submit_response_example?: Record<string, unknown>
  fetch_response_example?: Record<string, unknown>
}

type AsyncTaskParameterMappingConfig = {
  name?: string
  when?: Record<string, unknown>
  strip_fields?: string[]
  set_fields?: Record<string, unknown>
}

type AsyncTaskResponseMappingConfig = {
  submit_task_id_paths?: string[]
  submit_status_paths?: string[]
  fetch_status_paths?: string[]
  fetch_status_map?: Record<string, string>
  fetch_progress_paths?: string[]
  fetch_result_url_paths?: string[]
  fetch_error_message_paths?: string[]
}

type ModelEndpointTemplateConfig = {
  path?: string
  method?: string
  description?: string
  request_example?: Record<string, unknown>
  parameters?: ModelEndpointTemplateParameterConfig[]
}

type ModelEndpointTemplateParameterConfig = {
  name?: string
  type?: string
  default?: unknown
  range?: string
  enum_values?: string[]
  description?: string
  required?: boolean
}

type ModelEndpointTemplateDraft = {
  id: string
  key: string
  path: string
  method: string
  description: string
  requestExample: string
  parameters: string
}

type AsyncTaskUpstreamTemplateDraft = {
  id: string
  enabled: boolean
  name: string
  channelId: string
  channelType: string
  requestPath: string
  model: string
  nameRule: string
  submitPath: string
  fetchPath: string
  method: string
  headers: string
  stripFields: string
  setFields: string
  parameterMappings: string
  body: string
  responseMapping: string
  submitResponseExample: string
  fetchResponseExample: string
}

let asyncTaskTemplateId = 0
let modelEndpointTemplateId = 0

function createAsyncTaskTemplateId() {
  asyncTaskTemplateId += 1
  return `async_task_template_${asyncTaskTemplateId}`
}

function createModelEndpointTemplateId() {
  modelEndpointTemplateId += 1
  return `model_endpoint_template_${modelEndpointTemplateId}`
}

function parseModelEndpointTemplateDrafts(value: string) {
  try {
    const parsed = JSON.parse(normalizeJsonText(value, '{}'))
    if (parsed === null || Array.isArray(parsed) || typeof parsed !== 'object') {
      return []
    }
    return Object.entries(parsed as Record<string, ModelEndpointTemplateConfig>)
      .map(([key, template]) => ({
        id: createModelEndpointTemplateId(),
        key,
        path: template?.path ?? '',
        method: template?.method ?? 'POST',
        description: template?.description ?? '',
        requestExample: stringifyTemplatePart(template?.request_example, {}),
        parameters: stringifyTemplatePart(template?.parameters, []),
      }))
      .sort((a, b) => a.key.localeCompare(b.key))
  } catch {
    return []
  }
}

function serializeModelEndpointTemplateDrafts(
  drafts: ModelEndpointTemplateDraft[]
) {
  const templates: Record<string, ModelEndpointTemplateConfig> = {}
  for (const draft of drafts) {
    const key = draft.key.trim()
    if (!key) continue
    const template: ModelEndpointTemplateConfig = {
      path: draft.path.trim(),
      method: draft.method.trim().toUpperCase() || 'POST',
    }
    const description = draft.description.trim()
    if (description) template.description = description
    const requestExample = parseTemplateJsonPart<Record<string, unknown>>(
      draft.requestExample,
      {}
    )
    if (Object.keys(requestExample).length > 0) {
      template.request_example = requestExample
    }
    const parameters = parseTemplateJsonPart<
      ModelEndpointTemplateParameterConfig[]
    >(draft.parameters, [])
    if (parameters.length > 0) template.parameters = parameters
    templates[key] = template
  }
  return JSON.stringify(templates, null, 2)
}

function createEmptyModelEndpointTemplateDraft(): ModelEndpointTemplateDraft {
  return {
    id: createModelEndpointTemplateId(),
    key: '',
    path: '',
    method: 'POST',
    description: '',
    requestExample: stringifyTemplatePart(modelEndpointRequestExample, {}),
    parameters: stringifyTemplatePart(modelEndpointParametersExample, []),
  }
}

function stringifyTemplatePart(value: unknown, fallback: unknown) {
  const source = value === undefined || value === null ? fallback : value
  return JSON.stringify(source, null, 2)
}

function parseTemplateJsonPart<T>(value: string, fallback: T): T {
  const trimmed = value.trim()
  if (!trimmed) return fallback
  try {
    return JSON.parse(trimmed) as T
  } catch {
    return fallback
  }
}

function parseOptionalTemplateNumber(value: string): number | undefined {
  const trimmed = value.trim()
  if (!trimmed) return undefined
  const parsed = Number(trimmed)
  if (!Number.isFinite(parsed) || parsed <= 0) return undefined
  return parsed
}

function isSupportedPath(
  value: string,
  options: Array<{ value: string; label: string }>
) {
  return options.some((option) => option.value === value)
}

function getSupportedPathMethod(path: string) {
  return (
    supportedModelEndpointPaths.find((option) => option.value === path)
      ?.method ?? 'POST'
  )
}

function parseAsyncTaskTemplateDrafts(value: string) {
  try {
    const parsed = JSON.parse(normalizeJsonText(value, '[]'))
    if (!Array.isArray(parsed)) return []
    return parsed.map((item) => {
      const template = item as AsyncTaskUpstreamTemplateConfig
      return {
        id: createAsyncTaskTemplateId(),
        enabled: template.enabled === true,
        name: template.name ?? '',
        channelId: template.channel_id ? String(template.channel_id) : '',
        channelType: template.channel_type ? String(template.channel_type) : '',
        requestPath: template.request_path ?? '/v1/images/generations/async',
        model: template.model ?? '',
        nameRule: String(template.name_rule ?? 0),
        submitPath: template.submit_path ?? '',
        fetchPath: template.fetch_path ?? '',
        method: template.method ?? 'POST',
        headers: stringifyTemplatePart(template.headers, {}),
        stripFields: stringifyTemplatePart(template.strip_fields, []),
        setFields: stringifyTemplatePart(template.set_fields, {}),
        parameterMappings: stringifyTemplatePart(
          template.parameter_mappings,
          []
        ),
        body: stringifyTemplatePart(template.body, {}),
        responseMapping: stringifyTemplatePart(template.response_mapping, {}),
        submitResponseExample: stringifyTemplatePart(
          template.submit_response_example,
          {}
        ),
        fetchResponseExample: stringifyTemplatePart(
          template.fetch_response_example,
          {}
        ),
      } satisfies AsyncTaskUpstreamTemplateDraft
    })
  } catch {
    return []
  }
}

function serializeAsyncTaskTemplateDrafts(
  drafts: AsyncTaskUpstreamTemplateDraft[]
) {
  const templates = drafts.map((draft) => {
    const template: AsyncTaskUpstreamTemplateConfig = {
      enabled: draft.enabled,
    }

    const name = draft.name.trim()
    if (name) template.name = name

    const channelId = parseOptionalTemplateNumber(draft.channelId)
    if (channelId !== undefined) template.channel_id = channelId

    const channelType = parseOptionalTemplateNumber(draft.channelType)
    if (channelType !== undefined) template.channel_type = channelType

    const requestPath = draft.requestPath.trim()
    if (requestPath) template.request_path = requestPath

    const model = draft.model.trim()
    if (model) template.model = model

    const nameRule = Number(draft.nameRule)
    if (Number.isFinite(nameRule) && nameRule > 0) {
      template.name_rule = nameRule
    }

    const submitPath = draft.submitPath.trim()
    if (submitPath) template.submit_path = submitPath

    const fetchPath = draft.fetchPath.trim()
    if (fetchPath) template.fetch_path = fetchPath

    const method = draft.method.trim().toUpperCase()
    if (method && method !== 'POST') template.method = method

    const headers = parseTemplateJsonPart<Record<string, string>>(
      draft.headers,
      {}
    )
    if (Object.keys(headers).length > 0) template.headers = headers

    const stripFields = parseTemplateJsonPart<string[]>(draft.stripFields, [])
    if (stripFields.length > 0) template.strip_fields = stripFields

    const setFields = parseTemplateJsonPart<Record<string, unknown>>(
      draft.setFields,
      {}
    )
    if (Object.keys(setFields).length > 0) template.set_fields = setFields

    const parameterMappings = parseTemplateJsonPart<
      AsyncTaskParameterMappingConfig[]
    >(draft.parameterMappings, [])
    if (parameterMappings.length > 0) {
      template.parameter_mappings = parameterMappings
    }

    const body = parseTemplateJsonPart<Record<string, unknown>>(draft.body, {})
    if (Object.keys(body).length > 0) template.body = body

    const responseMapping =
      parseTemplateJsonPart<AsyncTaskResponseMappingConfig>(
        draft.responseMapping,
        {}
      )
    if (Object.keys(responseMapping).length > 0) {
      template.response_mapping = responseMapping
    }

    const submitResponseExample = parseTemplateJsonPart<Record<string, unknown>>(
      draft.submitResponseExample,
      {}
    )
    if (Object.keys(submitResponseExample).length > 0) {
      template.submit_response_example = submitResponseExample
    }

    const fetchResponseExample = parseTemplateJsonPart<Record<string, unknown>>(
      draft.fetchResponseExample,
      {}
    )
    if (Object.keys(fetchResponseExample).length > 0) {
      template.fetch_response_example = fetchResponseExample
    }

    return template
  })

  return JSON.stringify(templates, null, 2)
}

function createEmptyAsyncTaskTemplateDraft(): AsyncTaskUpstreamTemplateDraft {
  return {
    id: createAsyncTaskTemplateId(),
    enabled: false,
    name: '',
    channelId: '',
    channelType: '',
    requestPath: '/v1/images/generations/async',
    model: '',
    nameRule: '0',
    submitPath: '',
    fetchPath: '',
    method: 'POST',
    headers: stringifyTemplatePart(asyncTaskHeadersExample, {}),
    stripFields: stringifyTemplatePart(asyncTaskStripFieldsExample, []),
    setFields: stringifyTemplatePart(asyncTaskSetFieldsExample, {}),
    parameterMappings: stringifyTemplatePart(
      asyncTaskParameterMappingsExample,
      []
    ),
    body: stringifyTemplatePart(asyncTaskBodyExample, {}),
    responseMapping: JSON.stringify(
      defaultAsyncTaskResponseMapping,
      null,
      2
    ),
    submitResponseExample: stringifyTemplatePart(
      asyncTaskSubmitResponseExample,
      {}
    ),
    fetchResponseExample: stringifyTemplatePart(
      asyncTaskFetchResponseExample,
      {}
    ),
  }
}

function isTemplateJsonObject(value: string) {
  try {
    const parsed = JSON.parse(value.trim() || '{}')
    return parsed !== null && !Array.isArray(parsed) && typeof parsed === 'object'
  } catch {
    return false
  }
}

function isTemplateJsonArray(value: string) {
  try {
    return Array.isArray(JSON.parse(value.trim() || '[]'))
  } catch {
    return false
  }
}

function isAsyncTaskParameterMappingsJson(value: string) {
  try {
    const parsed = JSON.parse(value.trim() || '[]')
    if (!Array.isArray(parsed)) return false
    return parsed.every((item) => {
      if (item === null || Array.isArray(item) || typeof item !== 'object') {
        return false
      }
      const mapping = item as AsyncTaskParameterMappingConfig
      const when = mapping.when
      if (
        when === null ||
        Array.isArray(when) ||
        typeof when !== 'object' ||
        Object.keys(when).length === 0
      ) {
        return false
      }
      if (
        mapping.strip_fields !== undefined &&
        (!Array.isArray(mapping.strip_fields) ||
          !mapping.strip_fields.every((field) => typeof field === 'string'))
      ) {
        return false
      }
      if (
        mapping.set_fields !== undefined &&
        (mapping.set_fields === null ||
          Array.isArray(mapping.set_fields) ||
          typeof mapping.set_fields !== 'object')
      ) {
        return false
      }
      return (
        (mapping.strip_fields?.length ?? 0) > 0 ||
        Object.keys(mapping.set_fields ?? {}).length > 0
      )
    })
  } catch {
    return false
  }
}

function isAsyncTaskTemplateDraftValid(draft: AsyncTaskUpstreamTemplateDraft) {
  return (
    isTemplateJsonObject(draft.headers) &&
    isTemplateJsonArray(draft.stripFields) &&
    isTemplateJsonObject(draft.setFields) &&
    isAsyncTaskParameterMappingsJson(draft.parameterMappings) &&
    isTemplateJsonObject(draft.body) &&
    isTemplateJsonObject(draft.responseMapping) &&
    isTemplateJsonObject(draft.submitResponseExample) &&
    isTemplateJsonObject(draft.fetchResponseExample)
  )
}

function isModelEndpointTemplateDraftValid(draft: ModelEndpointTemplateDraft) {
  return (
    isTemplateJsonObject(draft.requestExample) &&
    isTemplateJsonArray(draft.parameters)
  )
}

function readJsonObject(value: string): Record<string, unknown> | undefined {
  try {
    const parsed = JSON.parse(value.trim() || '{}')
    if (parsed !== null && !Array.isArray(parsed) && typeof parsed === 'object') {
      return parsed as Record<string, unknown>
    }
  } catch {
    return undefined
  }
  return undefined
}

function readPathValue(source: unknown, path: string): unknown {
  const parts = path
    .split('.')
    .map((part) => part.trim())
    .filter(Boolean)
  let current = source
  for (const part of parts) {
    if (Array.isArray(current)) {
      const index = Number(part)
      if (!Number.isInteger(index)) return undefined
      current = current[index]
    } else if (current !== null && typeof current === 'object') {
      current = (current as Record<string, unknown>)[part]
    } else {
      return undefined
    }
  }
  return current
}

function toPreviewString(value: unknown): string {
  if (value === undefined || value === null) return ''
  if (typeof value === 'string') return value.trim()
  if (typeof value === 'number' || typeof value === 'boolean') {
    return String(value)
  }
  if (Array.isArray(value)) {
    for (const item of value) {
      const itemValue = toPreviewString(item)
      if (itemValue) return itemValue
    }
  }
  return ''
}

function readFirstPath(source: unknown, paths?: string[]) {
  for (const path of paths ?? []) {
    const value = toPreviewString(readPathValue(source, path))
    if (value) return value
  }
  return ''
}

function normalizePreviewStatus(
  status: string,
  statusMap?: Record<string, string>
) {
  const normalized = status.trim()
  if (!normalized) return ''
  for (const [key, value] of Object.entries(statusMap ?? {})) {
    if (key === normalized || key.toLowerCase() === normalized.toLowerCase()) {
      return value
    }
  }
  return normalized
}

function buildMappingPreview(draft: AsyncTaskUpstreamTemplateDraft) {
  const mapping = readJsonObject(
    draft.responseMapping
  ) as AsyncTaskResponseMappingConfig
  const submitResponse = readJsonObject(draft.submitResponseExample)
  const fetchResponse = readJsonObject(draft.fetchResponseExample)
  if (!mapping) {
    return {
      taskId: '',
      submitStatus: '',
      fetchStatus: '',
      progress: '',
      resultUrl: '',
      errorMessage: '',
    }
  }

  return {
    taskId: submitResponse
      ? readFirstPath(submitResponse, mapping.submit_task_id_paths)
      : '',
    submitStatus: submitResponse
      ? normalizePreviewStatus(
          readFirstPath(submitResponse, mapping.submit_status_paths),
          mapping.fetch_status_map
        )
      : '',
    fetchStatus: fetchResponse
      ? normalizePreviewStatus(
          readFirstPath(fetchResponse, mapping.fetch_status_paths),
          mapping.fetch_status_map
        )
      : '',
    progress: fetchResponse
      ? readFirstPath(fetchResponse, mapping.fetch_progress_paths)
      : '',
    resultUrl: fetchResponse
      ? readFirstPath(fetchResponse, mapping.fetch_result_url_paths)
      : '',
    errorMessage: fetchResponse
      ? readFirstPath(fetchResponse, mapping.fetch_error_message_paths)
      : '',
  }
}

export function EndpointTemplateSettingsCard({
  defaultValues,
}: EndpointTemplateSettingsCardProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [modelEndpointMode, setModelEndpointMode] = useState<
    'visual' | 'json'
  >('visual')
  const [asyncTemplateMode, setAsyncTemplateMode] = useState<'visual' | 'json'>(
    'visual'
  )
  const [guideOpen, setGuideOpen] = useState(false)
  const form = useForm<EndpointTemplateSettingsFormValues>({
    resolver: zodResolver(schema),
    defaultValues,
  })

  useEffect(() => {
    form.reset(defaultValues)
  }, [defaultValues, form])

  const formatField = (
    field: keyof EndpointTemplateSettingsFormValues,
    fallback: string
  ) => {
    try {
      form.setValue(field, formatJson(form.getValues(field), fallback), {
        shouldDirty: true,
      })
    } catch {
      toast.error(t('Invalid JSON format'))
    }
  }

  const copyExample = async (example: string) => {
    await navigator.clipboard.writeText(example)
    toast.success(t('Copied'))
  }

  const onSubmit = async (values: EndpointTemplateSettingsFormValues) => {
    const normalizedValues = {
      ModelEndpointTemplates: normalizeJsonText(
        values.ModelEndpointTemplates,
        '{}'
      ),
      AsyncTaskUpstreamTemplates: normalizeJsonText(
        values.AsyncTaskUpstreamTemplates,
        '[]'
      ),
    }
    const normalizedDefaults = {
      ModelEndpointTemplates: normalizeJsonText(
        defaultValues.ModelEndpointTemplates,
        '{}'
      ),
      AsyncTaskUpstreamTemplates: normalizeJsonText(
        defaultValues.AsyncTaskUpstreamTemplates,
        '[]'
      ),
    }
    const updates = Object.entries(normalizedValues).filter(
      ([key, value]) =>
        value !== normalizedDefaults[key as keyof typeof normalizedDefaults]
    )

    if (updates.length === 0) {
      toast.info(t('No changes to save'))
      return
    }

    for (const [key, value] of updates) {
      await updateOption.mutateAsync({ key, value })
    }
  }

  return (
    <SettingsSection title={t('Global Template Configuration')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />

          <div className='flex flex-wrap justify-end gap-2'>
            <Button
              type='button'
              variant='outline'
              size='sm'
              onClick={() => setGuideOpen(true)}
            >
              <HelpCircle data-icon='inline-start' />
              {t('Usage guide')}
            </Button>
          </div>
          <EndpointTemplateGuide open={guideOpen} onOpenChange={setGuideOpen} />

          <Tabs defaultValue='model-endpoints' className='gap-4'>
            <TabsList className='w-fit'>
              <TabsTrigger value='model-endpoints'>
                {t('Model endpoint templates')}
              </TabsTrigger>
              <TabsTrigger value='async-upstreams'>
                {t('Async task upstream templates')}
              </TabsTrigger>
            </TabsList>

            <TabsContent value='model-endpoints'>
              <FormField
                control={form.control}
                name='ModelEndpointTemplates'
                render={({ field }) => (
                  <FormItem>
                    <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
                      <div className='flex flex-col gap-1'>
                        <FormLabel>{t('Model endpoint templates')}</FormLabel>
                      </div>
                      <Button
                        type='button'
                        variant='outline'
                        size='sm'
                        onClick={() =>
                          setModelEndpointMode((mode) =>
                            mode === 'visual' ? 'json' : 'visual'
                          )
                        }
                      >
                        {modelEndpointMode === 'visual' ? (
                          <Code2 data-icon='inline-start' />
                        ) : (
                          <Palette data-icon='inline-start' />
                        )}
                        {modelEndpointMode === 'visual'
                          ? t('Switch to JSON')
                          : t('Switch to visual editor')}
                      </Button>
                    </div>
                    {modelEndpointMode === 'visual' ? (
                      <ModelEndpointTemplateVisualEditor
                        value={field.value}
                        onChange={field.onChange}
                      />
                    ) : (
                      <FormControl>
                        <Textarea
                          rows={14}
                          className='font-mono text-xs'
                          placeholder={modelEndpointTemplatesExample}
                          {...field}
                          onChange={(event) =>
                            field.onChange(event.target.value)
                          }
                        />
                      </FormControl>
                    )}
                    <div className='flex flex-wrap gap-2'>
                      <Button
                        type='button'
                        variant='outline'
                        size='sm'
                        onClick={() =>
                          form.setValue(
                            'ModelEndpointTemplates',
                            modelEndpointTemplatesExample,
                            { shouldDirty: true }
                          )
                        }
                      >
                        {t('Fill example')}
                      </Button>
                      <Button
                        type='button'
                        variant='outline'
                        size='sm'
                        onClick={() =>
                          formatField('ModelEndpointTemplates', '{}')
                        }
                      >
                        {t('Format JSON')}
                      </Button>
                      <Button
                        type='button'
                        variant='outline'
                        size='sm'
                        onClick={() =>
                          copyExample(modelEndpointTemplatesExample)
                        }
                      >
                        <Copy data-icon='inline-start' />
                        {t('Copy example')}
                      </Button>
                    </div>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </TabsContent>

            <TabsContent value='async-upstreams'>
              <FormField
                control={form.control}
                name='AsyncTaskUpstreamTemplates'
                render={({ field }) => (
                  <FormItem>
                    <div className='flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between'>
                      <div className='flex flex-col gap-1'>
                        <FormLabel>
                          {t('Async task upstream templates')}
                        </FormLabel>
                      </div>
                      <Button
                        type='button'
                        variant='outline'
                        size='sm'
                        onClick={() =>
                          setAsyncTemplateMode((mode) =>
                            mode === 'visual' ? 'json' : 'visual'
                          )
                        }
                      >
                        {asyncTemplateMode === 'visual' ? (
                          <Code2 data-icon='inline-start' />
                        ) : (
                          <Palette data-icon='inline-start' />
                        )}
                        {asyncTemplateMode === 'visual'
                          ? t('Switch to JSON')
                          : t('Switch to visual editor')}
                      </Button>
                    </div>
                    {asyncTemplateMode === 'visual' ? (
                      <AsyncTaskUpstreamTemplateVisualEditor
                        value={field.value}
                        onChange={field.onChange}
                      />
                    ) : (
                      <FormControl>
                        <Textarea
                          rows={18}
                          className='font-mono text-xs'
                          placeholder={asyncTaskUpstreamTemplatesExample}
                          {...field}
                          onChange={(event) =>
                            field.onChange(event.target.value)
                          }
                        />
                      </FormControl>
                    )}
                    <div className='flex flex-wrap gap-2'>
                      <Button
                        type='button'
                        variant='outline'
                        size='sm'
                        onClick={() =>
                          form.setValue(
                            'AsyncTaskUpstreamTemplates',
                            asyncTaskUpstreamTemplatesExample,
                            { shouldDirty: true }
                          )
                        }
                      >
                        {t('Fill example')}
                      </Button>
                      <Button
                        type='button'
                        variant='outline'
                        size='sm'
                        onClick={() =>
                          formatField('AsyncTaskUpstreamTemplates', '[]')
                        }
                      >
                        {t('Format JSON')}
                      </Button>
                      <Button
                        type='button'
                        variant='outline'
                        size='sm'
                        onClick={() =>
                          copyExample(asyncTaskUpstreamTemplatesExample)
                        }
                      >
                        <Copy data-icon='inline-start' />
                        {t('Copy example')}
                      </Button>
                    </div>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </TabsContent>
          </Tabs>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}

type EndpointTemplateGuideProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

function EndpointTemplateGuideCodeBlock({ children }: { children: string }) {
  return (
    <pre className='bg-muted/60 overflow-x-auto rounded-lg border px-3 py-2 text-xs leading-6 whitespace-pre-wrap'>
      {children}
    </pre>
  )
}

function EndpointTemplateGuide({
  open,
  onOpenChange,
}: EndpointTemplateGuideProps) {
  const { t } = useTranslation()

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side='right'
        className={sideDrawerContentClassName('sm:max-w-2xl')}
      >
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>{t('Template configuration usage guide')}</SheetTitle>
          <SheetDescription>
            {t(
              'Understand model endpoint templates and async upstream templates before enabling them.'
            )}
          </SheetDescription>
        </SheetHeader>

        <div className={sideDrawerFormClassName('gap-5')}>
          <section className='flex flex-col gap-2'>
            <h3 className='text-sm font-semibold'>
              {t('What this page controls')}
            </h3>
            <div className='text-muted-foreground flex flex-col gap-2 text-sm leading-6'>
              <p>
                {t(
                  'Model endpoint templates control the usage examples shown in model management and client documentation.'
                )}
              </p>
              <p>
                {t(
                  'Async task upstream templates rewrite upstream submit/fetch requests for existing NewAPI async routes.'
                )}
              </p>
              <p>
                {t(
                  'They do not create new public API paths. Request path must be one of the supported local routes.'
                )}
              </p>
            </div>
          </section>

          <Accordion className='rounded-lg border px-3'>
            <AccordionItem value='model-endpoint'>
              <AccordionTrigger>
                {t('Model endpoint template')}
              </AccordionTrigger>
              <AccordionContent className='flex flex-col gap-3'>
                <p className='text-muted-foreground text-sm leading-6'>
                  {t(
                    'Use this when a model needs to display a specific API path, request method, request example, and parameter list.'
                  )}
                </p>
                <EndpointTemplateGuideCodeBlock>
                  {`{
  "image-generation-async": {
    "path": "/v1/images/generations/async",
    "method": "POST",
    "request_example": {
      "model": "{model}",
      "prompt": "A koi pond at sunset",
      "size": "1024x1024"
    }
  }
}`}
                </EndpointTemplateGuideCodeBlock>
                <p className='text-muted-foreground text-sm leading-6'>
                  {t(
                    'Common placeholders include {model}, {endpoint_type}, {endpoint_path}, and {method}.'
                  )}
                </p>
              </AccordionContent>
            </AccordionItem>

            <AccordionItem value='async-upstream'>
              <AccordionTrigger>{t('Async upstream template')}</AccordionTrigger>
              <AccordionContent className='flex flex-col gap-3'>
                <p className='text-muted-foreground text-sm leading-6'>
                  {t(
                    'Use this when the local route stays the same, but a selected upstream channel needs a different submit path, fetch path, or body format.'
                  )}
                </p>
                <EndpointTemplateGuideCodeBlock>
                  {`{
  "enabled": true,
  "channel_id": 19,
  "request_path": "/v1/images/generations/async",
  "model": "gpt-image-2",
  "submit_path": "/v1/videos",
  "fetch_path": "/v1/videos/{task_id}"
}`}
                </EndpointTemplateGuideCodeBlock>
                <p className='text-muted-foreground text-sm leading-6'>
                  {t(
                    'Enabled templates are checked from top to bottom. The first matching channel, request path, and model rule is used.'
                  )}
                </p>
                <p className='text-muted-foreground text-sm leading-6'>
                  {t(
                    'Channel ID has priority when configured. Channel type can be used as a broader fallback. Leave model empty to match all models.'
                  )}
                </p>
              </AccordionContent>
            </AccordionItem>

            <AccordionItem value='body'>
              <AccordionTrigger>{t('Request body rewriting')}</AccordionTrigger>
              <AccordionContent className='flex flex-col gap-3'>
                <p className='text-muted-foreground text-sm leading-6'>
                  {t(
                    'Body JSON rebuilds the whole upstream request body. Fields not listed there are not sent upstream.'
                  )}
                </p>
                <p className='text-muted-foreground text-sm leading-6'>
                  {t(
                    'Strip fields JSON removes fields from the selected body before set fields are applied.'
                  )}
                </p>
                <p className='text-muted-foreground text-sm leading-6'>
                  {t(
                    'Set fields JSON adds or overrides fields after strip fields. It is suitable for changing model, prompt, size, or nested parameters.'
                  )}
                </p>
                <EndpointTemplateGuideCodeBlock>
                  {`"strip_fields": ["metadata.debug"],
"body": {
  "model": "{upstream_model}",
  "prompt": "{body:prompt}",
  "size": "{body:size}"
},
"set_fields": {
  "parameters.ratio": "{body:ratio}"
}`}
                </EndpointTemplateGuideCodeBlock>
              </AccordionContent>
            </AccordionItem>

            <AccordionItem value='parameter-mapping'>
              <AccordionTrigger>
                {t('Request parameter mapping')}
              </AccordionTrigger>
              <AccordionContent className='flex flex-col gap-3'>
                <p className='text-muted-foreground text-sm leading-6'>
                  {t(
                    'Parameter mappings inspect the original client JSON request and conditionally remove, add, or override fields sent upstream.'
                  )}
                </p>
                <p className='text-muted-foreground text-sm leading-6'>
                  {t(
                    'Rules are checked from top to bottom and stop after the first match. Every field in when must match exactly, including its JSON type and letter case.'
                  )}
                </p>
                <p className='text-muted-foreground text-sm leading-6'>
                  {t(
                    'Field paths support nested objects such as metadata.resolution and parameters.size. This mapping currently applies to JSON async requests only.'
                  )}
                </p>
                <EndpointTemplateGuideCodeBlock>
                  {`[
  {
    "name": "1:1 2K to 2048x2048",
    "when": { "size": "1:1", "resolution": "2k" },
    "strip_fields": ["resolution"],
    "set_fields": { "size": "2048x2048" }
  }
]`}
                </EndpointTemplateGuideCodeBlock>
              </AccordionContent>
            </AccordionItem>

            <AccordionItem value='mapping'>
              <AccordionTrigger>{t('Response mapping')}</AccordionTrigger>
              <AccordionContent className='flex flex-col gap-3'>
                <p className='text-muted-foreground text-sm leading-6'>
                  {t(
                    'Response mapping tells the system where to read task ID, status, progress, result URL, and error message from upstream responses.'
                  )}
                </p>
                <p className='text-muted-foreground text-sm leading-6'>
                  {t(
                    'fetch_status_map only needs provider-specific values. Add more mappings only when the upstream status is not already normalized.'
                  )}
                </p>
                <EndpointTemplateGuideCodeBlock>
                  {`{
  "submit_task_id_paths": ["id", "data.id"],
  "fetch_status_paths": ["status", "data.status"],
  "fetch_status_map": {
    "completed": "SUCCESS",
    "failed": "FAILURE"
  },
  "fetch_result_url_paths": ["url", "data.result.images.0.url"]
}`}
                </EndpointTemplateGuideCodeBlock>
              </AccordionContent>
            </AccordionItem>

            <AccordionItem value='advice'>
              <AccordionTrigger>{t('Operational advice')}</AccordionTrigger>
              <AccordionContent className='flex flex-col gap-2'>
                <p className='text-muted-foreground text-sm leading-6'>
                  {t('Keep new templates disabled until you finish a test request.')}
                </p>
                <p className='text-muted-foreground text-sm leading-6'>
                  {t('Put provider-specific templates before broad fallback templates.')}
                </p>
                <p className='text-muted-foreground text-sm leading-6'>
                  {t(
                    'After changing templates, test submit and fetch flows with one model and one channel first.'
                  )}
                </p>
              </AccordionContent>
            </AccordionItem>
          </Accordion>
        </div>
      </SheetContent>
    </Sheet>
  )
}

type ModelEndpointTemplateVisualEditorProps = {
  value: string
  onChange: (value: string) => void
}

function ModelEndpointTemplateVisualEditor({
  value,
  onChange,
}: ModelEndpointTemplateVisualEditorProps) {
  const { t } = useTranslation()
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set())
  const [drafts, setDrafts] = useState<ModelEndpointTemplateDraft[]>(() =>
    parseModelEndpointTemplateDrafts(value)
  )

  useEffect(() => {
    setDrafts((currentDrafts) => {
      if (
        currentDrafts.every(isModelEndpointTemplateDraftValid) &&
        serializeModelEndpointTemplateDrafts(currentDrafts) ===
        normalizeJsonText(value, '{}')
      ) {
        return currentDrafts
      }
      return parseModelEndpointTemplateDrafts(value)
    })
  }, [value])

  const emitDrafts = (nextDrafts: ModelEndpointTemplateDraft[]) => {
    setDrafts(nextDrafts)
    if (nextDrafts.every(isModelEndpointTemplateDraftValid)) {
      onChange(serializeModelEndpointTemplateDrafts(nextDrafts))
    }
  }

  const updateDraft = (
    id: string,
    field: keyof Omit<ModelEndpointTemplateDraft, 'id'>,
    nextValue: string
  ) => {
    emitDrafts(
      drafts.map((draft) =>
        draft.id === id ? { ...draft, [field]: nextValue } : draft
      )
    )
  }

  const updateTemplatePath = (draft: ModelEndpointTemplateDraft, path: string) => {
    emitDrafts(
      drafts.map((item) =>
        item.id === draft.id
          ? { ...item, path, method: getSupportedPathMethod(path) }
          : item
      )
    )
  }

  const addDraft = () => {
    const draft = createEmptyModelEndpointTemplateDraft()
    emitDrafts([...drafts, draft])
    setExpandedIds((current) => new Set(current).add(draft.id))
  }

  const removeDraft = (id: string) => {
    emitDrafts(drafts.filter((draft) => draft.id !== id))
    setExpandedIds((current) => {
      const next = new Set(current)
      next.delete(id)
      return next
    })
  }

  const toggleExpanded = (id: string) => {
    setExpandedIds((current) => {
      const next = new Set(current)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  const duplicateKeys = drafts
    .map((draft) => draft.key.trim())
    .filter((key, index, keys) => key && keys.indexOf(key) !== index)

  return (
    <div className='flex flex-col gap-3'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <p className='text-muted-foreground text-sm'>
          {t(
            'Controls the endpoint templates available in the model management form.'
          )}
        </p>
        <Button type='button' variant='outline' size='sm' onClick={addDraft}>
          <Plus data-icon='inline-start' />
          {t('Add template')}
        </Button>
      </div>

      {drafts.length === 0 ? (
        <div className='rounded-lg border p-6 text-center text-sm text-muted-foreground'>
          {t('No model endpoint templates configured')}
        </div>
      ) : (
        <div className='flex flex-col gap-3'>
          {drafts.map((draft) => {
            const duplicated = duplicateKeys.includes(draft.key.trim())
            const expanded = expandedIds.has(draft.id)
            return (
              <div key={draft.id} className='rounded-lg border bg-background'>
                <div className='flex flex-col gap-3 border-b p-3 lg:flex-row lg:items-center'>
                  <div className='flex min-w-0 flex-1 items-center gap-2'>
                    <Button
                      type='button'
                      variant='ghost'
                      size='sm'
                      onClick={() => toggleExpanded(draft.id)}
                      aria-label={
                        expanded
                          ? t('Collapse configuration')
                          : t('Expand configuration')
                      }
                    >
                      <ChevronRight
                        data-icon='inline-start'
                        className={expanded ? 'rotate-90' : ''}
                      />
                    </Button>
                    <Input
                      className='min-w-0 flex-1'
                      value={draft.key}
                      placeholder='image-generation-async'
                      aria-invalid={duplicated}
                      onChange={(event) =>
                        updateDraft(draft.id, 'key', event.target.value)
                      }
                    />
                  </div>
                  <Button
                    type='button'
                    variant='ghost'
                    size='sm'
                    className='self-end lg:self-auto'
                    onClick={() => removeDraft(draft.id)}
                    aria-label={t('Delete')}
                  >
                    <Trash2 data-icon='inline-start' />
                  </Button>
                </div>

                <div className='grid gap-3 p-3 md:grid-cols-2'>
                  <TemplateInlineField label={t('Request path')}>
                    <NativeSelect
                      className='w-full'
                      value={draft.path}
                      onChange={(event) =>
                        updateTemplatePath(draft, event.target.value)
                      }
                    >
                      {draft.path &&
                        !isSupportedPath(
                          draft.path,
                          supportedModelEndpointPaths
                        ) && (
                          <NativeSelectOption value={draft.path}>
                            {t('Unsupported custom path: {{path}}', {
                              path: draft.path,
                            })}
                          </NativeSelectOption>
                        )}
                      {supportedModelEndpointPaths.map((option) => (
                        <NativeSelectOption
                          key={option.value}
                          value={option.value}
                        >
                          {t(option.label)} - {option.value}
                        </NativeSelectOption>
                      ))}
                    </NativeSelect>
                  </TemplateInlineField>
                  <TemplateInlineField label={t('Method')}>
                    <NativeSelect
                      className='w-full'
                      value={draft.method}
                      onChange={(event) =>
                        updateDraft(draft.id, 'method', event.target.value)
                      }
                    >
                      <NativeSelectOption value='POST'>POST</NativeSelectOption>
                      <NativeSelectOption value='GET'>GET</NativeSelectOption>
                    </NativeSelect>
                  </TemplateInlineField>
                </div>

                {expanded && (
                  <div className='border-t bg-muted/20 p-3'>
                    <div className='flex flex-col gap-3 rounded-lg border bg-background p-3'>
                      <TemplateInlineField label={t('Endpoint description')}>
                        <Textarea
                          rows={3}
                          value={draft.description}
                          onChange={(event) =>
                            updateDraft(
                              draft.id,
                              'description',
                              event.target.value
                            )
                          }
                        />
                      </TemplateInlineField>
                      <div className='grid gap-3 md:grid-cols-2'>
                        <TemplateJsonField
                          label={t('Request example JSON')}
                          value={draft.requestExample}
                          invalid={!isTemplateJsonObject(draft.requestExample)}
                          onChange={(nextValue) =>
                            updateDraft(
                              draft.id,
                              'requestExample',
                              nextValue
                            )
                          }
                        />
                        <TemplateJsonField
                          label={t('Parameters JSON')}
                          value={draft.parameters}
                          invalid={!isTemplateJsonArray(draft.parameters)}
                          onChange={(nextValue) =>
                            updateDraft(draft.id, 'parameters', nextValue)
                          }
                        />
                      </div>
                    </div>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}

      {duplicateKeys.length > 0 && (
        <p className='text-destructive text-sm'>
          {t('Duplicate template keys: {{keys}}', {
            keys: Array.from(new Set(duplicateKeys)).join(', '),
          })}
        </p>
      )}
    </div>
  )
}

type AsyncTaskUpstreamTemplateVisualEditorProps = {
  value: string
  onChange: (value: string) => void
}

function formatChannelOptionLabel(channel: Channel) {
  return `#${channel.id} ${channel.name || '-'}`
}

function parseTemplateModelList(value: string) {
  return value
    .split(',')
    .map((model) => model.trim())
    .filter(Boolean)
}

function AsyncTaskUpstreamTemplateVisualEditor({
  value,
  onChange,
}: AsyncTaskUpstreamTemplateVisualEditorProps) {
  const { t } = useTranslation()
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set())
  const [drafts, setDrafts] = useState<AsyncTaskUpstreamTemplateDraft[]>(() =>
    parseAsyncTaskTemplateDrafts(value)
  )
  const { data: channelsResponse } = useQuery({
    queryKey: ['async-task-upstream-template-channels'],
    queryFn: () =>
      getChannels({
        p: 1,
        page_size: 1000,
        id_sort: true,
      }),
    staleTime: 60_000,
  })
  const channels = channelsResponse?.data?.items ?? []
  const channelsById = new Map(channels.map((channel) => [channel.id, channel]))

  useEffect(() => {
    setDrafts((currentDrafts) => {
      if (
        currentDrafts.every(isAsyncTaskTemplateDraftValid) &&
        serializeAsyncTaskTemplateDrafts(currentDrafts) ===
          normalizeJsonText(value, '[]')
      ) {
        return currentDrafts
      }
      return parseAsyncTaskTemplateDrafts(value)
    })
  }, [value])

  const emitDrafts = (nextDrafts: AsyncTaskUpstreamTemplateDraft[]) => {
    setDrafts(nextDrafts)
    if (nextDrafts.every(isAsyncTaskTemplateDraftValid)) {
      onChange(serializeAsyncTaskTemplateDrafts(nextDrafts))
    }
  }

  const updateDraft = (
    id: string,
    field: keyof Omit<AsyncTaskUpstreamTemplateDraft, 'id'>,
    nextValue: string | boolean
  ) => {
    emitDrafts(
      drafts.map((draft) =>
        draft.id === id ? { ...draft, [field]: nextValue } : draft
      )
    )
  }

  const updateChannelId = (
    draft: AsyncTaskUpstreamTemplateDraft,
    value: string
  ) => {
    const selectedChannel = channelsById.get(Number(value))
    emitDrafts(
      drafts.map((item) =>
        item.id === draft.id
          ? {
              ...item,
              channelId: value,
              channelType: selectedChannel
                ? String(selectedChannel.type)
                : item.channelType,
            }
          : item
      )
    )
  }

  const addDraft = () => {
    const draft = createEmptyAsyncTaskTemplateDraft()
    emitDrafts([...drafts, draft])
    setExpandedIds((current) => new Set(current).add(draft.id))
  }

  const removeDraft = (id: string) => {
    emitDrafts(drafts.filter((draft) => draft.id !== id))
    setExpandedIds((current) => {
      const next = new Set(current)
      next.delete(id)
      return next
    })
  }

  const toggleExpanded = (id: string) => {
    setExpandedIds((current) => {
      const next = new Set(current)
      if (next.has(id)) {
        next.delete(id)
      } else {
        next.add(id)
      }
      return next
    })
  }

  return (
    <div className='flex flex-col gap-3'>
      <div className='flex flex-wrap items-center justify-between gap-2'>
        <p className='text-muted-foreground text-sm'>
          {t(
            'Only enabled templates participate in async channel matching. Disabled templates are saved but never affect routing.'
          )}
        </p>
        <Button type='button' variant='outline' size='sm' onClick={addDraft}>
          <Plus data-icon='inline-start' />
          {t('Add template')}
        </Button>
      </div>

      {drafts.length === 0 ? (
        <div className='rounded-lg border p-6 text-center text-sm text-muted-foreground'>
          {t('No async upstream templates configured')}
        </div>
      ) : (
        <div className='flex flex-col gap-3'>
          {drafts.map((draft) => {
            const expanded = expandedIds.has(draft.id)
            const mappingPreview = buildMappingPreview(draft)
            const selectedChannel = channelsById.get(Number(draft.channelId))
            const selectedModels = parseTemplateModelList(draft.model)
            const modelOptions = Array.from(
              new Set([
                ...parseTemplateModelList(selectedChannel?.models ?? ''),
                ...selectedModels,
              ])
            ).map<MultiSelectOption>((model) => ({
              label: model,
              value: model,
            }))
            return (
              <div key={draft.id} className='rounded-lg border bg-background'>
                <div className='flex flex-col gap-3 border-b p-3 lg:flex-row lg:items-center'>
                  <div className='flex min-w-0 flex-1 items-center gap-2'>
                    <Button
                      type='button'
                      variant='ghost'
                      size='sm'
                      onClick={() => toggleExpanded(draft.id)}
                      aria-label={
                        expanded
                          ? t('Collapse configuration')
                          : t('Expand configuration')
                      }
                    >
                      <ChevronRight
                        data-icon='inline-start'
                        className={expanded ? 'rotate-90' : ''}
                      />
                    </Button>
                    <Switch
                      checked={draft.enabled}
                      onCheckedChange={(checked) =>
                        updateDraft(draft.id, 'enabled', checked)
                      }
                      aria-label={t('Enabled')}
                    />
                    <Input
                      className='min-w-0 flex-1'
                      value={draft.name}
                      placeholder={t('Template name')}
                      onChange={(event) =>
                        updateDraft(draft.id, 'name', event.target.value)
                      }
                    />
                  </div>
                  <Button
                    type='button'
                    variant='ghost'
                    size='sm'
                    className='self-end lg:self-auto'
                    onClick={() => removeDraft(draft.id)}
                    aria-label={t('Delete')}
                  >
                    <Trash2 data-icon='inline-start' />
                  </Button>
                </div>

                <div
                  hidden={!expanded}
                  className='flex flex-col gap-3 bg-muted/20 p-3'
                >
                  <TemplateSection title={t('Channel and path configuration')}>
                    <div className='grid gap-3 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4'>
                  <TemplateInlineField label={t('Channel ID')}>
                    <NativeSelect
                      className='w-full'
                      value={draft.channelId}
                      onChange={(event) =>
                        updateChannelId(draft, event.target.value)
                      }
                    >
                      <NativeSelectOption value=''>
                        {t('Any channel')}
                      </NativeSelectOption>
                      {draft.channelId &&
                        !channelsById.has(Number(draft.channelId)) && (
                          <NativeSelectOption value={draft.channelId}>
                            #{draft.channelId}
                          </NativeSelectOption>
                        )}
                      {channels.map((channel) => (
                        <NativeSelectOption
                          key={channel.id}
                          value={String(channel.id)}
                        >
                          {formatChannelOptionLabel(channel)}
                        </NativeSelectOption>
                      ))}
                    </NativeSelect>
                  </TemplateInlineField>
                  <TemplateInlineField label={t('Channel type')}>
                    <NativeSelect
                      className='w-full'
                      value={draft.channelType}
                      onChange={(event) =>
                        updateDraft(
                          draft.id,
                          'channelType',
                          event.target.value
                        )
                      }
                    >
                      <NativeSelectOption value=''>
                        {t('Any channel type')}
                      </NativeSelectOption>
                      {draft.channelType &&
                        !CHANNEL_TYPE_OPTIONS.some(
                          (option) => option.value === Number(draft.channelType)
                        ) && (
                          <NativeSelectOption value={draft.channelType}>
                            #{draft.channelType}
                          </NativeSelectOption>
                        )}
                      {CHANNEL_TYPE_OPTIONS.map((option) => (
                        <NativeSelectOption
                          key={option.value}
                          value={String(option.value)}
                        >
                          #{option.value} {t(option.label)}
                        </NativeSelectOption>
                      ))}
                    </NativeSelect>
                  </TemplateInlineField>
                  <TemplateInlineField label={t('Request path')}>
                    <NativeSelect
                      className='w-full'
                      value={draft.requestPath}
                      onChange={(event) =>
                        updateDraft(
                          draft.id,
                          'requestPath',
                          event.target.value
                        )
                      }
                    >
                      {draft.requestPath &&
                        !isSupportedPath(
                          draft.requestPath,
                          supportedAsyncRequestPaths
                        ) && (
                          <NativeSelectOption value={draft.requestPath}>
                            {t('Unsupported custom path: {{path}}', {
                              path: draft.requestPath,
                            })}
                          </NativeSelectOption>
                        )}
                      {supportedAsyncRequestPaths.map((option) => (
                        <NativeSelectOption
                          key={option.value}
                          value={option.value}
                        >
                          {t(option.label)} - {option.value}
                        </NativeSelectOption>
                      ))}
                    </NativeSelect>
                  </TemplateInlineField>
                  <TemplateInlineField label={t('Model')}>
                    <MultiSelect
                      options={modelOptions}
                      selected={selectedModels}
                      onChange={(models) =>
                        updateDraft(draft.id, 'model', models.join(','))
                      }
                      placeholder={t('Select models')}
                      emptyText={t('No models available')}
                      maxVisibleChips={4}
                    />
                  </TemplateInlineField>
                  <TemplateInlineField label={t('Match rule')}>
                    <NativeSelect
                      className='w-full'
                      value={draft.nameRule}
                      onChange={(event) =>
                        updateDraft(draft.id, 'nameRule', event.target.value)
                      }
                    >
                      <NativeSelectOption value='0'>
                        {t('Exact Match')}
                      </NativeSelectOption>
                      <NativeSelectOption value='1'>
                        {t('Prefix Match')}
                      </NativeSelectOption>
                      <NativeSelectOption value='2'>
                        {t('Contains Match')}
                      </NativeSelectOption>
                      <NativeSelectOption value='3'>
                        {t('Suffix Match')}
                      </NativeSelectOption>
                    </NativeSelect>
                  </TemplateInlineField>
                  <TemplateInlineField label={t('Submit path')}>
                    <Input
                      value={draft.submitPath}
                      placeholder='/v1/videos'
                      onChange={(event) =>
                        updateDraft(draft.id, 'submitPath', event.target.value)
                      }
                    />
                  </TemplateInlineField>
                  <TemplateInlineField label={t('Fetch path')}>
                    <Input
                      value={draft.fetchPath}
                      placeholder='/v1/videos/{task_id}'
                      onChange={(event) =>
                        updateDraft(draft.id, 'fetchPath', event.target.value)
                      }
                    />
                  </TemplateInlineField>
                  <TemplateInlineField label={t('Submit method')}>
                    <NativeSelect
                      className='w-full'
                      value={draft.method}
                      onChange={(event) =>
                        updateDraft(draft.id, 'method', event.target.value)
                      }
                    >
                      <NativeSelectOption value='POST'>POST</NativeSelectOption>
                      <NativeSelectOption value='GET'>GET</NativeSelectOption>
                    </NativeSelect>
                  </TemplateInlineField>
                    </div>
                  </TemplateSection>
                </div>

                {expanded && (
                  <div className='border-t bg-muted/20 p-3'>
                    <div className='flex flex-col gap-3'>
                      <TemplateSection title={t('Request configuration')}>
                        <div className='flex flex-col gap-3'>
                        <TemplateJsonField
                          label={t('Headers JSON')}
                          value={draft.headers}
                          invalid={!isTemplateJsonObject(draft.headers)}
                          onChange={(nextValue) =>
                            updateDraft(draft.id, 'headers', nextValue)
                          }
                        />
                        <TemplateJsonField
                          label={t('Strip fields JSON')}
                          value={draft.stripFields}
                          invalid={!isTemplateJsonArray(draft.stripFields)}
                          onChange={(nextValue) =>
                            updateDraft(draft.id, 'stripFields', nextValue)
                          }
                        />
                        <TemplateJsonField
                          label={t('Set fields JSON')}
                          value={draft.setFields}
                          invalid={!isTemplateJsonObject(draft.setFields)}
                          onChange={(nextValue) =>
                            updateDraft(draft.id, 'setFields', nextValue)
                          }
                        />
                        <TemplateJsonField
                          label={t('Body JSON')}
                          value={draft.body}
                          invalid={!isTemplateJsonObject(draft.body)}
                          onChange={(nextValue) =>
                            updateDraft(draft.id, 'body', nextValue)
                          }
                        />
                          <TemplateJsonField
                            label={t('Request parameter mappings JSON')}
                            value={draft.parameterMappings}
                            rows={10}
                            invalid={
                              !isAsyncTaskParameterMappingsJson(
                                draft.parameterMappings
                              )
                            }
                            onChange={(nextValue) =>
                              updateDraft(
                                draft.id,
                                'parameterMappings',
                                nextValue
                              )
                            }
                          />
                        </div>
                      </TemplateSection>
                      <TemplateSection title={t('Response mapping')}>
                        <div className='flex flex-col gap-3'>
                        <TemplateJsonField
                          label={t('Response mapping JSON')}
                          value={draft.responseMapping}
                          invalid={!isTemplateJsonObject(draft.responseMapping)}
                          onChange={(nextValue) =>
                            updateDraft(
                              draft.id,
                              'responseMapping',
                              nextValue
                            )
                          }
                        />
                        <div className='flex flex-col gap-3'>
                          <TemplateJsonField
                            label={t('Submit response example JSON')}
                            value={draft.submitResponseExample}
                            invalid={
                              !isTemplateJsonObject(
                                draft.submitResponseExample
                              )
                            }
                            onChange={(nextValue) =>
                              updateDraft(
                                draft.id,
                                'submitResponseExample',
                                nextValue
                              )
                            }
                          />
                          <TemplateJsonField
                            label={t('Fetch response example JSON')}
                            value={draft.fetchResponseExample}
                            invalid={
                              !isTemplateJsonObject(draft.fetchResponseExample)
                            }
                            onChange={(nextValue) =>
                              updateDraft(
                                draft.id,
                                'fetchResponseExample',
                                nextValue
                              )
                            }
                          />
                        </div>
                        <div className='grid gap-2 rounded-md border bg-background p-3 text-xs md:grid-cols-2 xl:grid-cols-3'>
                          <MappingPreviewItem
                            label={t('Upstream task ID')}
                            value={mappingPreview.taskId}
                          />
                          <MappingPreviewItem
                            label={t('Submit status')}
                            value={mappingPreview.submitStatus}
                          />
                          <MappingPreviewItem
                            label={t('Fetch status')}
                            value={mappingPreview.fetchStatus}
                          />
                          <MappingPreviewItem
                            label={t('Progress')}
                            value={mappingPreview.progress}
                          />
                          <MappingPreviewItem
                            label={t('Result URL')}
                            value={mappingPreview.resultUrl}
                          />
                          <MappingPreviewItem
                            label={t('Error message')}
                            value={mappingPreview.errorMessage}
                          />
                        </div>
                        </div>
                      </TemplateSection>
                    </div>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

type MappingPreviewItemProps = {
  label: string
  value: string
}

type TemplateSectionProps = {
  title: string
  children: ReactNode
}

type TemplateSectionContextValue = {
  expandVersion: number
}

const TemplateSectionContext = createContext<TemplateSectionContextValue | null>(
  null
)

function TemplateSection({ title, children }: TemplateSectionProps) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(true)
  const [expandVersion, setExpandVersion] = useState(1)

  const handleOpenChange = (nextExpanded: boolean) => {
    setExpanded(nextExpanded)
    if (nextExpanded) setExpandVersion((version) => version + 1)
  }

  return (
    <Collapsible
      open={expanded}
      onOpenChange={handleOpenChange}
      className='rounded-lg border bg-background p-3'
    >
      <div className='flex items-center justify-between gap-2'>
        <span className='text-sm font-medium'>{title}</span>
        <CollapsibleTrigger
          render={
            <Button
              type='button'
              variant='secondary'
              size='sm'
              className='min-w-20 font-medium'
            />
          }
        >
          {expanded ? t('Collapse') : t('Expand')}
          <ChevronRight
            data-icon='inline-end'
            aria-hidden='true'
            className={cn(
              'transition-transform duration-200',
              expanded && 'rotate-90'
            )}
          />
        </CollapsibleTrigger>
      </div>
      <CollapsibleContent className='mt-3'>
        <TemplateSectionContext.Provider value={{ expandVersion }}>
          {children}
        </TemplateSectionContext.Provider>
      </CollapsibleContent>
    </Collapsible>
  )
}

function MappingPreviewItem({ label, value }: MappingPreviewItemProps) {
  const { t } = useTranslation()
  return (
    <div className='flex min-w-0 flex-col gap-1'>
      <span className='text-muted-foreground'>{label}</span>
      <span className='truncate font-mono'>{value || t('Not resolved')}</span>
    </div>
  )
}

type TemplateInlineFieldProps = {
  label: string
  children: ReactNode
}

function TemplateInlineField({ label, children }: TemplateInlineFieldProps) {
  return (
    <label className='flex min-w-0 flex-col gap-1.5'>
      <span className='text-muted-foreground text-xs'>{label}</span>
      {children}
    </label>
  )
}

type TemplateJsonFieldProps = {
  label: string
  value: string
  invalid: boolean
  rows?: number
  onChange: (value: string) => void
}

function TemplateJsonField({
  label,
  value,
  invalid,
  rows = 4,
  onChange,
}: TemplateJsonFieldProps) {
  const { t } = useTranslation()
  const [expanded, setExpanded] = useState(false)
  const sectionContext = useContext(TemplateSectionContext)

  useEffect(() => {
    if (sectionContext?.expandVersion) setExpanded(true)
  }, [sectionContext?.expandVersion])

  return (
    <Collapsible
      open={expanded}
      onOpenChange={setExpanded}
      className='flex min-w-0 flex-col gap-1.5'
    >
      <div className='flex items-center justify-between gap-2'>
        <span className='text-muted-foreground text-xs'>{label}</span>
        <CollapsibleTrigger
          render={
            <Button
              type='button'
              variant='ghost'
              size='xs'
              className='text-muted-foreground min-w-16 font-normal'
            />
          }
        >
          {expanded ? t('Collapse') : t('Expand')}
          <ChevronRight
            data-icon='inline-end'
            aria-hidden='true'
            className={cn(
              'transition-transform duration-200',
              expanded && 'rotate-90'
            )}
          />
        </CollapsibleTrigger>
      </div>
      <CollapsibleContent className='flex flex-col gap-1.5'>
        <Textarea
          rows={rows}
          className='font-mono text-xs'
          aria-invalid={invalid}
          value={value}
          onChange={(event) => onChange(event.target.value)}
        />
      </CollapsibleContent>
      {invalid && (
        <span className='text-destructive text-xs'>
          {t('Invalid JSON format')}
        </span>
      )}
    </Collapsible>
  )
}
