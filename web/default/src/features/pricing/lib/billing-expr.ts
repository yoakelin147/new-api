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
/**
 * Billing expression parsing utilities.
 *
 * Mirrors the parser used by the classic frontend so that the dynamic
 * pricing breakdown UI can be rendered from the same backend expressions.
 *
 * The grammar is intentionally narrow: we only support the shapes that the
 * server emits (tiered pricing + request-rule conditional multipliers), so
 * the regular expressions are exact rather than tolerant of arbitrary
 * expression syntax.
 */

// ---------------------------------------------------------------------------
// Variable registry
// ---------------------------------------------------------------------------

export type BillingVar = {
  key: string
  field: string | null
  tierField: string | null
  label: string
  shortLabel: string
  side: 'input' | 'output' | 'condition'
  isBase?: boolean
  isConditionOnly?: boolean
  group?: string
}

export const BILLING_VARS: BillingVar[] = [
  {
    key: 'p',
    field: 'inputPrice',
    tierField: 'input_unit_cost',
    label: 'Input price',
    shortLabel: 'Input',
    side: 'input',
    isBase: true,
  },
  {
    key: 'c',
    field: 'outputPrice',
    tierField: 'output_unit_cost',
    label: 'Completion price',
    shortLabel: 'Output',
    side: 'output',
    isBase: true,
  },
  {
    key: 'len',
    field: null,
    tierField: null,
    label: 'Input length',
    shortLabel: 'Length',
    side: 'condition',
    isConditionOnly: true,
  },
  {
    key: 'cr',
    field: 'cacheReadPrice',
    tierField: 'cache_read_unit_cost',
    label: 'Cache read price',
    shortLabel: 'Cache Read',
    side: 'input',
    group: 'cache',
  },
  {
    key: 'cc',
    field: 'cacheCreatePrice',
    tierField: 'cache_create_unit_cost',
    label: 'Cache create price',
    shortLabel: 'Cache Write',
    side: 'input',
    group: 'cache',
  },
  {
    key: 'cc1h',
    field: 'cacheCreate1hPrice',
    tierField: 'cache_create_1h_unit_cost',
    label: 'Cache create (1h) price',
    shortLabel: 'Cache Write (1h)',
    side: 'input',
    group: 'cache',
  },
  {
    key: 'img',
    field: 'imagePrice',
    tierField: 'image_unit_cost',
    label: 'Image input price',
    shortLabel: 'Image In',
    side: 'input',
    group: 'media',
  },
  {
    key: 'img_o',
    field: 'imageOutputPrice',
    tierField: 'image_output_unit_cost',
    label: 'Image output price',
    shortLabel: 'Image Out',
    side: 'output',
    group: 'media',
  },
  {
    key: 'ai',
    field: 'audioInputPrice',
    tierField: 'audio_input_unit_cost',
    label: 'Audio input price',
    shortLabel: 'Audio In',
    side: 'input',
    group: 'media',
  },
  {
    key: 'ao',
    field: 'audioOutputPrice',
    tierField: 'audio_output_unit_cost',
    label: 'Audio output price',
    shortLabel: 'Audio Out',
    side: 'output',
    group: 'media',
  },
]

/** Vars that have real price fields (excludes condition-only vars like `len`) */
export const BILLING_PRICING_VARS: BillingVar[] = BILLING_VARS.filter(
  (v) => !v.isConditionOnly
)

/** Vars valid in tier conditions (`p`, `c`, `len`) */
export const BILLING_CONDITION_VARS: string[] = BILLING_VARS.filter(
  (v) => v.isBase || v.isConditionOnly
).map((v) => v.key)

const BILLING_VAR_KEY_TO_FIELD = Object.fromEntries(
  BILLING_PRICING_VARS.map((v) => [v.key, v.field as string])
) as Record<string, string>

export const BILLING_EXTRA_VARS: BillingVar[] = BILLING_VARS.filter(
  (v) => !v.isBase && !v.isConditionOnly
)

export const BILLING_CACHE_VAR_MAP = BILLING_EXTRA_VARS.map((v) => ({
  field: v.tierField as string,
  exprVar: v.key,
}))

const BILLING_VAR_REGEX = new RegExp(
  `\\b(${BILLING_PRICING_VARS.map((v) => v.key).join('|')})\\s*\\*\\s*([\\d.eE+-]+)`,
  'g'
)

// ---------------------------------------------------------------------------
// Request rule constants
// ---------------------------------------------------------------------------

export const SOURCE_PARAM = 'param'
export const SOURCE_HEADER = 'header'
export const SOURCE_TIME = 'time'

export const MATCH_EQ = 'eq'
export const MATCH_CONTAINS = 'contains'
export const MATCH_GT = 'gt'
export const MATCH_GTE = 'gte'
export const MATCH_LT = 'lt'
export const MATCH_LTE = 'lte'
export const MATCH_EXISTS = 'exists'
export const MATCH_RANGE = 'range'

export const TIME_FUNCS = ['hour', 'minute', 'weekday', 'month', 'day'] as const
export type TimeFunc = (typeof TIME_FUNCS)[number]

export const COMMON_TIMEZONES: { value: string; label: string }[] = [
  { value: 'Asia/Shanghai', label: 'UTC+8 Shanghai (Asia/Shanghai)' },
  { value: 'UTC', label: 'UTC' },
  { value: 'America/New_York', label: 'UTC-5 New York (America/New_York)' },
  {
    value: 'America/Los_Angeles',
    label: 'UTC-8 Los Angeles (America/Los_Angeles)',
  },
  { value: 'America/Chicago', label: 'UTC-6 Chicago (America/Chicago)' },
  { value: 'Europe/London', label: 'UTC+0 London (Europe/London)' },
  { value: 'Europe/Berlin', label: 'UTC+1 Berlin (Europe/Berlin)' },
  { value: 'Asia/Tokyo', label: 'UTC+9 Tokyo (Asia/Tokyo)' },
  { value: 'Asia/Singapore', label: 'UTC+8 Singapore (Asia/Singapore)' },
  { value: 'Asia/Seoul', label: 'UTC+9 Seoul (Asia/Seoul)' },
  { value: 'Australia/Sydney', label: 'UTC+10 Sydney (Australia/Sydney)' },
]

const NUMERIC_LITERAL_REGEX = /^-?(?:\d+\.?\d*|\.\d+)(?:[eE][+-]?\d+)?$/

export type ParamHeaderCondition = {
  source: 'param' | 'header'
  path: string
  mode: string
  value: string
}

export type TimeCondition = {
  source: 'time'
  timeFunc: TimeFunc
  timezone: string
  mode: string
  value: string
  rangeStart: string
  rangeEnd: string
}

export type RequestCondition = TimeCondition | ParamHeaderCondition

export type RequestRuleGroup = {
  conditions: RequestCondition[]
  multiplier: string
}

export type TierCondition = {
  var: 'p' | 'c' | 'len'
  op: '<' | '<=' | '>' | '>='
  value: number
}

export type ParsedTier = {
  label: string
  conditions: TierCondition[]
  conditionText?: string
  isFallback?: boolean
  [field: string]: unknown
}

export type TimeTierPrice = {
  label: string
  inputPrice: number
  outputPrice: number
  cacheReadPrice: number
}

export type TimeTierVisualConfig = {
  timezone: string
  weekdayStart: number
  weekdayEnd: number
  ranges: { start: number; end: number }[]
  matched: TimeTierPrice
  fallback: TimeTierPrice
}

// ---------------------------------------------------------------------------
// Tier parser
// ---------------------------------------------------------------------------

function stripExprVersion(exprStr: string): { version: number; body: string } {
  if (!exprStr) return { version: 1, body: '' }
  const m = exprStr.match(/^v(\d+):([\s\S]*)$/)
  if (m) return { version: Number(m[1]), body: m[2] }
  return { version: 1, body: exprStr }
}

function parseTierBody(bodyStr: string): Record<string, number> | null {
  const coeffs: Record<string, number> = {}
  const re = new RegExp(BILLING_VAR_REGEX.source, 'g')
  let end = 0
  let m
  while ((m = re.exec(bodyStr)) !== null) {
    const separator = bodyStr.slice(end, m.index).trim()
    const value = Number(m[2])
    if (
      separator !== (end === 0 ? '' : '+') ||
      !NUMERIC_LITERAL_REGEX.test(m[2]) ||
      !Number.isFinite(value) ||
      value < 0 ||
      Object.hasOwn(coeffs, m[1])
    ) {
      return null
    }
    coeffs[m[1]] = value
    end = re.lastIndex
  }
  if (end === 0 || bodyStr.slice(end).trim()) return null
  const tier: Record<string, number> = {}
  for (const [varName, field] of Object.entries(BILLING_VAR_KEY_TO_FIELD)) {
    if (Object.hasOwn(coeffs, varName)) tier[field] = coeffs[varName]
  }
  return tier
}

function findTopLevelTernary(
  expression: string
): { condition: string; yes: string; no: string } | null {
  let depth = 0
  let quoted = false
  let escaped = false
  let questionIndex = -1
  let nestedTernaries = 0

  for (let index = 0; index < expression.length; index += 1) {
    const char = expression[index]
    if (quoted) {
      if (escaped) escaped = false
      else if (char === '\\') escaped = true
      else if (char === '"') quoted = false
      continue
    }
    if (char === '"') {
      quoted = true
      continue
    }
    if (char === '(') depth += 1
    else if (char === ')') depth -= 1
    else if (depth === 0 && char === '?') {
      if (questionIndex < 0) questionIndex = index
      else nestedTernaries += 1
    } else if (depth === 0 && char === ':' && questionIndex >= 0) {
      if (nestedTernaries > 0) {
        nestedTernaries -= 1
        continue
      }
      return {
        condition: expression.slice(0, questionIndex).trim(),
        yes: expression.slice(questionIndex + 1, index).trim(),
        no: expression.slice(index + 1).trim(),
      }
    }
  }
  return null
}

function parseNamedTier(
  expression: string,
  conditionText?: string,
  isFallback = false
): ParsedTier | null {
  const match = unwrapOuterParens(expression.trim()).match(
    /^tier\(\s*("(?:[^"\\]|\\.)*")\s*,\s*([\s\S]+)\)$/
  )
  if (!match) return null
  const prices = parseTierBody(match[2])
  if (!prices) return null
  const label = JSON.parse(match[1]) as string
  return {
    ...prices,
    label,
    conditions: [],
    ...(conditionText ? { conditionText } : {}),
    ...(isFallback ? { isFallback: true } : {}),
  }
}

function parseTimeConditionalTiers(expression: string): ParsedTier[] | null {
  const ternary = findTopLevelTernary(unwrapOuterParens(expression.trim()))
  if (
    !ternary ||
    !/\b(?:hour|minute|weekday|month|day)\s*\(/.test(ternary.condition)
  ) {
    return null
  }
  const conditionText = ternary.condition.replace(/\s+/g, ' ').trim()
  const matched = parseNamedTier(ternary.yes, conditionText)
  const fallback = parseNamedTier(ternary.no, undefined, true)
  return matched && fallback ? [fallback, matched] : null
}

function parseTimeSchedule(
  condition: string
): Pick<
  TimeTierVisualConfig,
  'timezone' | 'weekdayStart' | 'weekdayEnd' | 'ranges'
> | null {
  const calls = [...condition.matchAll(/\b(?:hour|weekday)\("([^"]+)"\)/g)]
  const timezone = calls[0]?.[1]
  if (!timezone || calls.some((call) => call[1] !== timezone)) return null
  const escapedTimezone = timezone.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  const weekdayStartMatch = condition.match(
    new RegExp(`weekday\\("${escapedTimezone}"\\)\\s*>=\\s*(\\d+)`)
  )
  const weekdayEndMatch = condition.match(
    new RegExp(`weekday\\("${escapedTimezone}"\\)\\s*<\\s*(\\d+)`)
  )
  if (!weekdayStartMatch || !weekdayEndMatch) return null
  const ranges = [
    ...condition.matchAll(
      /hour\("([^"]+)"\)\s*>=\s*(\d+)\s*&&\s*hour\("\1"\)\s*<\s*(\d+)/g
    ),
  ].map((match) => ({ start: Number(match[2]), end: Number(match[3]) }))
  if (ranges.length === 0) return null
  const remainder = condition
    .replace(/weekday\("[^"]+"\)\s*>=\s*\d+/g, '')
    .replace(/weekday\("[^"]+"\)\s*<\s*\d+/g, '')
    .replace(/hour\("([^"]+)"\)\s*>=\s*\d+\s*&&\s*hour\("\1"\)\s*<\s*\d+/g, '')
    .replace(/[()\s&|]/g, '')
  if (remainder) return null
  return {
    timezone,
    weekdayStart: Number(weekdayStartMatch[1]),
    weekdayEnd: Number(weekdayEndMatch[1]),
    ranges,
  }
}

function parsedTierToTimePrice(tier: ParsedTier): TimeTierPrice | null {
  const allowedFields = new Set(['inputPrice', 'outputPrice', 'cacheReadPrice'])
  const unsupported = BILLING_PRICING_VARS.some(
    (variable) =>
      variable.field &&
      tier[variable.field] !== undefined &&
      !allowedFields.has(variable.field)
  )
  if (unsupported) return null
  return {
    label: tier.label,
    inputPrice: Number(tier.inputPrice ?? 0),
    outputPrice: Number(tier.outputPrice ?? 0),
    cacheReadPrice: Number(tier.cacheReadPrice ?? 0),
  }
}

export function tryParseTimeTierVisualConfig(
  expression: string
): TimeTierVisualConfig | null {
  const ternary = findTopLevelTernary(unwrapOuterParens(expression.trim()))
  if (!ternary) return null
  const schedule = parseTimeSchedule(ternary.condition)
  const matchedTier = parseNamedTier(ternary.yes)
  const fallbackTier = parseNamedTier(ternary.no)
  if (!schedule || !matchedTier || !fallbackTier) return null
  const matched = parsedTierToTimePrice(matchedTier)
  const fallback = parsedTierToTimePrice(fallbackTier)
  return matched && fallback ? { ...schedule, matched, fallback } : null
}

function validExpressionNumber(value: number): number {
  return Number.isFinite(value) && value >= 0 ? value : 0
}

function buildTimeTierPrice(price: TimeTierPrice): string {
  return `tier(${JSON.stringify(price.label)}, p * ${validExpressionNumber(price.inputPrice)} + c * ${validExpressionNumber(price.outputPrice)} + cr * ${validExpressionNumber(price.cacheReadPrice)})`
}

export function generateTimeTierExpr(config: TimeTierVisualConfig): string {
  const timezone = JSON.stringify(config.timezone || 'UTC')
  const ranges = config.ranges
    .map(
      (range) =>
        `(hour(${timezone}) >= ${range.start} && hour(${timezone}) < ${range.end})`
    )
    .join(' || ')
  const condition = `weekday(${timezone}) >= ${config.weekdayStart} && weekday(${timezone}) < ${config.weekdayEnd} && (${ranges})`
  return `${condition} ? ${buildTimeTierPrice(config.matched)} : ${buildTimeTierPrice(config.fallback)}`
}

export function parseTiersFromExpr(exprStr: string): ParsedTier[] {
  if (!exprStr) return []
  try {
    const { body } = stripExprVersion(exprStr)
    const timeTiers = parseTimeConditionalTiers(body)
    if (timeTiers) return timeTiers
    const condGroup =
      `((?:(?:p|c|len)\\s*(?:<|<=|>|>=)\\s*[\\d.eE+]+)` +
      `(?:\\s*&&\\s*(?:p|c|len)\\s*(?:<|<=|>|>=)\\s*[\\d.eE+]+)*)`
    const tierRe = new RegExp(
      `(?:${condGroup}\\s*\\?\\s*)?tier\\("([^"]*)",\\s*([^)]+)\\)`,
      'g'
    )
    const tiers: ParsedTier[] = []
    let end = 0
    let m
    while ((m = tierRe.exec(body)) !== null) {
      if (body.slice(end, m.index).trim() !== (end === 0 ? '' : ':')) return []
      if (tiers.length > 0 && tiers.at(-1)?.conditions.length === 0) return []
      const condStr = m[1] || ''
      const conditions: TierCondition[] = []
      if (condStr) {
        for (const cp of condStr.split(/\s*&&\s*/)) {
          const cm = cp.trim().match(/^(p|c|len)\s*(<|<=|>|>=)\s*([\d.eE+]+)$/)
          if (cm) {
            conditions.push({
              var: cm[1] as TierCondition['var'],
              op: cm[2] as TierCondition['op'],
              value: Number(cm[3]),
            })
          }
        }
      }
      const tier = parseTierBody(m[3]) as ParsedTier | null
      if (!tier) return []
      tier.label = m[2]
      tier.conditions = conditions
      tiers.push(tier)
      end = tierRe.lastIndex
    }
    if (body.slice(end).trim() || tiers.at(-1)?.conditions.length) return []
    return tiers
  } catch {
    return []
  }
}

export function normalizeTierLabel(label: string | undefined): string {
  if (!label) return ''
  return label
    .replace(/<[=＝]?|≤|＜[=＝]?/g, '<')
    .replace(/>[=＝]?|≥|＞[=＝]?/g, '>')
    .replace(/\s+/g, '')
    .toLowerCase()
}

// ---------------------------------------------------------------------------
// Request rule parser
// ---------------------------------------------------------------------------

function splitTopLevelOperator(expr: string, operator: '*' | '&&'): string[] {
  const parts: string[] = []
  let start = 0
  let depth = 0
  let quoted = false
  let escaped = false
  for (let index = 0; index < expr.length; index += 1) {
    const char = expr[index]
    if (quoted) {
      if (escaped) {
        escaped = false
      } else if (char === '\\') {
        escaped = true
      } else if (char === '"') {
        quoted = false
      }
      continue
    }
    if (char === '"') {
      quoted = true
      continue
    }
    if (char === '(') depth += 1
    if (char === ')') depth -= 1
    if (
      depth === 0 &&
      expr.slice(index, index + operator.length) === operator
    ) {
      parts.push(expr.slice(start, index).trim())
      start = index + operator.length
      index += operator.length - 1
    }
  }
  parts.push(expr.slice(start).trim())
  return parts.filter(Boolean)
}

function splitTopLevelMultiply(expr: string): string[] {
  return splitTopLevelOperator(expr, '*')
}

function splitTopLevelAnd(expr: string): string[] {
  return splitTopLevelOperator(expr, '&&')
}

function parseExprLiteral(raw: string): string | null {
  const text = raw.trim()
  if (text === 'true' || text === 'false') return text
  if (NUMERIC_LITERAL_REGEX.test(text)) return text
  try {
    return JSON.parse(text) as string
  } catch {
    return null
  }
}

const TIME_FUNC_RANGES: Record<TimeFunc, [number, number]> = {
  hour: [0, 23],
  minute: [0, 59],
  weekday: [0, 6],
  month: [1, 12],
  day: [1, 31],
}

function isTimeValueInRange(timeFunc: TimeFunc, text: string): boolean {
  if (!NUMERIC_LITERAL_REGEX.test(text)) return false
  const value = Number(text)
  if (!Number.isInteger(value)) return false
  const [min, max] = TIME_FUNC_RANGES[timeFunc]
  return value >= min && value <= max
}

function tryParseTimeCondition(expr: string): RequestCondition | null {
  let m = expr.match(
    /^(hour|minute|weekday|month|day)\("([^"]+)"\) >= ([\d.eE+-]+) (&&|\|\|) \1\("\2"\) < ([\d.eE+-]+)$/
  )
  if (!m) {
    m = expr.match(
      /^\((hour|minute|weekday|month|day)\("([^"]+)"\) >= ([\d.eE+-]+) (&&|\|\|) \1\("\2"\) < ([\d.eE+-]+)\)$/
    )
  }
  if (m) {
    if (
      !isTimeValueInRange(m[1] as TimeFunc, m[3]) ||
      !isTimeValueInRange(m[1] as TimeFunc, m[5]) ||
      (Number(m[3]) > Number(m[5]) && m[4] !== '||')
    ) {
      return null
    }
    return {
      source: 'time',
      timeFunc: m[1] as TimeFunc,
      timezone: m[2],
      mode: MATCH_RANGE,
      value: '',
      rangeStart: m[3],
      rangeEnd: m[5],
    }
  }
  m = expr.match(
    /^(hour|minute|weekday|month|day)\("([^"]+)"\) (==|>=|<) ([\d.eE+-]+)$/
  )
  if (m) {
    if (!isTimeValueInRange(m[1] as TimeFunc, m[4])) return null
    const opMap: Record<string, string> = {
      '==': MATCH_EQ,
      '>=': MATCH_GTE,
      '<': MATCH_LT,
    }
    return {
      source: 'time',
      timeFunc: m[1] as TimeFunc,
      timezone: m[2],
      mode: opMap[m[3]] || MATCH_EQ,
      value: m[4],
      rangeStart: '',
      rangeEnd: '',
    }
  }
  return null
}

function tryParseTimeRangePair(
  lower: string,
  upper: string
): RequestCondition | null {
  const a = tryParseTimeCondition(lower)
  const b = tryParseTimeCondition(upper)
  if (!a || !b || a.source !== SOURCE_TIME || b.source !== SOURCE_TIME) {
    return null
  }
  if (a.timeFunc !== b.timeFunc || a.timezone !== b.timezone) return null
  if (a.mode !== MATCH_GTE || b.mode !== MATCH_LT) return null
  if (Number(a.value) > Number(b.value)) return null
  return {
    source: SOURCE_TIME,
    timeFunc: a.timeFunc,
    timezone: a.timezone,
    mode: MATCH_RANGE,
    value: '',
    rangeStart: a.value,
    rangeEnd: b.value,
  }
}

function tryParseRequestConditions(
  conditionStr: string
): RequestCondition[] | null {
  const wholeTimeCondition = tryParseTimeCondition(conditionStr.trim())
  if (wholeTimeCondition) return [wholeTimeCondition]

  const parts = splitTopLevelAnd(conditionStr)
  const conditions: RequestCondition[] = []
  for (let index = 0; index < parts.length; index += 1) {
    const part = parts[index].trim()
    const next = index + 1 < parts.length ? parts[index + 1].trim() : ''
    const range = next ? tryParseTimeRangePair(part, next) : null
    if (range) {
      conditions.push(range)
      index += 1
      continue
    }
    const condition = tryParseRequestCondition(part)
    if (!condition) return null
    conditions.push(condition)
  }
  return conditions.length > 0 ? conditions : null
}

function tryParseRequestCondition(expr: string): RequestCondition | null {
  const tc = tryParseTimeCondition(expr)
  if (tc) return tc

  let m = expr.match(/^header\("([^"]+)"\) != ""$/)
  if (m) return { source: 'header', path: m[1], mode: MATCH_EXISTS, value: '' }

  m = expr.match(/^param\("([^"]+)"\) != nil$/)
  if (m) return { source: 'param', path: m[1], mode: MATCH_EXISTS, value: '' }

  m = expr.match(/^has\(header\("([^"]+)"\), ((?:"(?:[^"\\]|\\.)*"))\)$/)
  if (m)
    return {
      source: 'header',
      path: m[1],
      mode: MATCH_CONTAINS,
      value: JSON.parse(m[2]) as string,
    }

  m = expr.match(
    /^param\("([^"]+)"\) != nil && has\(param\("([^"]+)"\), ((?:"(?:[^"\\]|\\.)*"))\)$/
  )
  if (m && m[1] === m[2])
    return {
      source: 'param',
      path: m[1],
      mode: MATCH_CONTAINS,
      value: JSON.parse(m[3]) as string,
    }

  m = expr.match(
    /^param\("([^"]+)"\) != nil && param\("([^"]+)"\) (>|>=|<|<=) ([\d.eE+-]+)$/
  )
  if (m && m[1] === m[2]) {
    const opMap: Record<string, string> = {
      '>': MATCH_GT,
      '>=': MATCH_GTE,
      '<': MATCH_LT,
      '<=': MATCH_LTE,
    }
    return { source: 'param', path: m[1], mode: opMap[m[3]], value: m[4] }
  }

  m = expr.match(/^(param|header)\("([^"]+)"\) == (.+)$/)
  if (m) {
    const parsedValue = parseExprLiteral(m[3])
    if (parsedValue === null) return null
    return {
      source: m[1] as 'param' | 'header',
      path: m[2],
      mode: MATCH_EQ,
      value: String(parsedValue),
    }
  }

  return null
}

function tryParseRuleGroupFactor(part: string): RequestRuleGroup | null {
  const m = part.match(/^\((.+) \? ([\d.eE+-]+) : 1\)$/s)
  if (!m) return null

  const conditionStr = m[1]
  const multiplier = m[2]

  const conditions = tryParseRequestConditions(conditionStr)
  if (!conditions) return null
  return { conditions, multiplier }
}

export function tryParseRequestRuleExpr(
  expr: string
): RequestRuleGroup[] | null {
  const trimmed = (expr || '').trim()
  if (!trimmed) return []

  const parts = splitTopLevelMultiply(trimmed)
  const groups: RequestRuleGroup[] = []
  for (const part of parts) {
    const group = tryParseRuleGroupFactor(part)
    if (!group) return null
    groups.push(group)
  }
  return groups
}

// ---------------------------------------------------------------------------
// Combine / split billing expr and request rules
// ---------------------------------------------------------------------------

function hasFullOuterParens(expr: string): boolean {
  if (!expr.startsWith('(') || !expr.endsWith(')')) return false
  let depth = 0
  for (let i = 0; i < expr.length; i += 1) {
    if (expr[i] === '(') depth += 1
    if (expr[i] === ')') depth -= 1
    if (depth === 0 && i < expr.length - 1) return false
  }
  return depth === 0
}

function unwrapOuterParens(expr: string): string {
  let current = (expr || '').trim()
  while (hasFullOuterParens(current)) {
    current = current.slice(1, -1).trim()
  }
  return current
}

export function splitBillingExprAndRequestRules(expr: string): {
  billingExpr: string
  requestRuleExpr: string
} {
  const trimmed = (expr || '').trim()
  if (!trimmed) return { billingExpr: '', requestRuleExpr: '' }

  const parts = splitTopLevelMultiply(trimmed)
  if (parts.length <= 1) return { billingExpr: trimmed, requestRuleExpr: '' }

  const ruleParts: string[] = []
  const baseParts: string[] = []

  parts.forEach((part) => {
    const parsed = tryParseRequestRuleExpr(part)
    if (parsed && parsed.length > 0) {
      ruleParts.push(part)
    } else {
      baseParts.push(part)
    }
  })

  if (ruleParts.length === 0 || baseParts.length !== 1) {
    return { billingExpr: trimmed, requestRuleExpr: '' }
  }

  return {
    billingExpr: unwrapOuterParens(baseParts[0]),
    requestRuleExpr: ruleParts.join(' * '),
  }
}

export function combineBillingExpr(
  baseExpr: string,
  requestRuleExpr: string
): string {
  const base = (baseExpr || '').trim()
  const rules = (requestRuleExpr || '').trim()
  if (!base) return ''
  if (!rules) return base
  return `(${base}) * ${rules}`
}

// ---------------------------------------------------------------------------
// Editor: empty constructors
// ---------------------------------------------------------------------------

export function createEmptyCondition(): ParamHeaderCondition {
  return { source: 'param', path: '', mode: MATCH_EQ, value: '' }
}

export function createEmptyTimeCondition(): TimeCondition {
  return {
    source: 'time',
    timeFunc: 'hour',
    timezone: 'Asia/Shanghai',
    mode: MATCH_GTE,
    value: '',
    rangeStart: '',
    rangeEnd: '',
  }
}

export function createEmptyRuleGroup(): RequestRuleGroup {
  return { conditions: [createEmptyCondition()], multiplier: '' }
}

export function createEmptyTimeRuleGroup(): RequestRuleGroup {
  return { conditions: [createEmptyTimeCondition()], multiplier: '' }
}

// ---------------------------------------------------------------------------
// Editor: match option helpers
// ---------------------------------------------------------------------------

export type MatchOption = { value: string; labelKey: string }

export function getRequestRuleMatchOptions(source: string): MatchOption[] {
  if (source === SOURCE_TIME) {
    return [
      { value: MATCH_EQ, labelKey: 'Equals' },
      { value: MATCH_GTE, labelKey: 'Greater than or equal' },
      { value: MATCH_LT, labelKey: 'Less than' },
      { value: MATCH_RANGE, labelKey: 'Overnight range' },
    ]
  }
  const base: MatchOption[] = [
    { value: MATCH_EQ, labelKey: 'Equals' },
    { value: MATCH_CONTAINS, labelKey: 'Contains' },
    { value: MATCH_EXISTS, labelKey: 'Exists' },
  ]
  if (source === SOURCE_HEADER) return base
  return [
    ...base,
    { value: MATCH_GT, labelKey: 'Greater than' },
    { value: MATCH_GTE, labelKey: 'Greater than or equal' },
    { value: MATCH_LT, labelKey: 'Less than' },
    { value: MATCH_LTE, labelKey: 'Less than or equal' },
  ]
}

// ---------------------------------------------------------------------------
// Editor: normalize a single condition
// ---------------------------------------------------------------------------

function isTimeFunc(value: unknown): value is TimeFunc {
  return typeof value === 'string' && TIME_FUNCS.includes(value as TimeFunc)
}

export function normalizeCondition(
  cond: Partial<RequestCondition> | null | undefined
): RequestCondition {
  const source =
    cond?.source === 'time'
      ? 'time'
      : cond?.source === 'header'
        ? 'header'
        : 'param'

  if (source === 'time') {
    const timeCond = cond as Partial<TimeCondition> | null | undefined
    const timeFunc: TimeFunc = isTimeFunc(timeCond?.timeFunc)
      ? timeCond.timeFunc
      : 'hour'
    const options = getRequestRuleMatchOptions(SOURCE_TIME)
    const mode = options.some((item) => item.value === timeCond?.mode)
      ? (timeCond?.mode as string)
      : MATCH_GTE
    return {
      source: 'time',
      timeFunc,
      timezone: timeCond?.timezone || 'Asia/Shanghai',
      mode,
      value: timeCond?.value == null ? '' : String(timeCond.value),
      rangeStart:
        timeCond?.rangeStart == null ? '' : String(timeCond.rangeStart),
      rangeEnd: timeCond?.rangeEnd == null ? '' : String(timeCond.rangeEnd),
    }
  }

  const phCond = cond as Partial<ParamHeaderCondition> | null | undefined
  const options = getRequestRuleMatchOptions(source)
  const mode = options.some((item) => item.value === phCond?.mode)
    ? (phCond?.mode as string)
    : MATCH_EQ
  return {
    source,
    path: phCond?.path || '',
    mode,
    value: phCond?.value == null ? '' : String(phCond.value),
  }
}

// ---------------------------------------------------------------------------
// Editor: build expression strings
// ---------------------------------------------------------------------------

function buildExprLiteral(mode: string, value: string): string {
  const text = String(value || '').trim()
  if (mode === MATCH_CONTAINS) return JSON.stringify(text)
  if (text === 'true' || text === 'false') return text
  if (NUMERIC_LITERAL_REGEX.test(text)) return text
  return JSON.stringify(text)
}

function buildTimeConditionExpr(cond: TimeCondition): string {
  const normalized = normalizeCondition(cond) as TimeCondition
  const { timeFunc, timezone, mode } = normalized
  const tz = JSON.stringify(timezone)
  const fn = `${timeFunc}(${tz})`

  if (mode === MATCH_RANGE) {
    const s = normalized.rangeStart.trim()
    const e = normalized.rangeEnd.trim()
    if (!isTimeValueInRange(timeFunc, s) || !isTimeValueInRange(timeFunc, e)) {
      return ''
    }
    return Number(s) > Number(e)
      ? `${fn} >= ${s} || ${fn} < ${e}`
      : `${fn} >= ${s} && ${fn} < ${e}`
  }
  const v = normalized.value.trim()
  if (!isTimeValueInRange(timeFunc, v)) return ''
  const opMap: Record<string, string> = {
    [MATCH_EQ]: '==',
    [MATCH_GTE]: '>=',
    [MATCH_LT]: '<',
  }
  return `${fn} ${opMap[mode] || '=='} ${v}`
}

function buildRequestConditionExpr(cond: RequestCondition): string {
  if (cond.source === 'time') return buildTimeConditionExpr(cond)
  const normalized = normalizeCondition(cond) as ParamHeaderCondition
  const path = normalized.path.trim()
  if (!path) return ''

  const sourceExpr =
    normalized.source === 'header'
      ? `header(${JSON.stringify(path)})`
      : `param(${JSON.stringify(path)})`

  switch (normalized.mode) {
    case MATCH_EXISTS:
      return normalized.source === 'header'
        ? `${sourceExpr} != ""`
        : `${sourceExpr} != nil`
    case MATCH_CONTAINS:
      return normalized.source === 'header'
        ? `has(${sourceExpr}, ${buildExprLiteral(normalized.mode, normalized.value)})`
        : `${sourceExpr} != nil && has(${sourceExpr}, ${buildExprLiteral(normalized.mode, normalized.value)})`
    case MATCH_GT:
    case MATCH_GTE:
    case MATCH_LT:
    case MATCH_LTE: {
      const opMap: Record<string, string> = {
        [MATCH_GT]: '>',
        [MATCH_GTE]: '>=',
        [MATCH_LT]: '<',
        [MATCH_LTE]: '<=',
      }
      const numText = String(normalized.value).trim()
      if (!NUMERIC_LITERAL_REGEX.test(numText)) return ''
      return `${sourceExpr} != nil && ${sourceExpr} ${opMap[normalized.mode]} ${numText}`
    }
    case MATCH_EQ:
    default:
      return `${sourceExpr} == ${buildExprLiteral(normalized.mode, normalized.value)}`
  }
}

function buildRuleGroupFactor(group: RequestRuleGroup): string {
  const multiplier = (group.multiplier || '').trim()
  if (!NUMERIC_LITERAL_REGEX.test(multiplier)) return ''
  const conditions = group.conditions || []
  const condExprs = conditions.map(buildRequestConditionExpr)
  if (condExprs.some((condition) => !condition)) return ''
  if (condExprs.length === 0) return ''

  const combined =
    condExprs.length === 1
      ? condExprs[0]
      : condExprs.map((e) => (e.includes(' || ') ? `(${e})` : e)).join(' && ')
  return `(${combined} ? ${multiplier} : 1)`
}

export function buildRequestRuleExpr(groups: RequestRuleGroup[]): string {
  const factors = (groups || []).map(buildRuleGroupFactor)
  if (factors.some((factor) => !factor)) return ''
  return factors.join(' * ')
}
