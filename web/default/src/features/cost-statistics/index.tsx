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
import { useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  CalendarDays,
  ChevronDown,
  ChevronsUpDown,
  Clock3,
  Download,
  Eye,
  KeyRound,
  RotateCcw,
  Search,
  Trash2,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import {
  formatLogQuota,
  formatNumber,
  formatTimestampForInput,
  formatTimestampToDate,
  formatTokens,
  parseTimestampFromInput,
  quotaUnitsToDollars,
} from '@/lib/format'
import { ROLE } from '@/lib/roles'
import { cn } from '@/lib/utils'
import { useIsAdmin } from '@/hooks/use-admin'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyTitle,
} from '@/components/ui/empty'
import { Input } from '@/components/ui/input'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { DataFreshnessIndicator } from '@/components/data-freshness-indicator'
import { StaticDataTable } from '@/components/data-table/static/static-data-table'
import { Main } from '@/components/layout'
import { SearchableSubjectSelect } from '@/components/searchable-subject-select'
import { getManagedLogUsers } from '@/features/usage-logs/api'
import { ModelBadge } from '@/features/usage-logs/components/model-badge'
import { getCostStatistics } from './api'
import {
  MAX_NON_ADMIN_CUSTOM_RANGE_SECONDS,
  validateCostStatisticsCustomRange,
} from './time-range'
import type {
  CostStatisticsData,
  CostStatisticsFilters,
  CostStatisticsPeriod,
  CostStatisticsRow,
  CostStatisticsSubject,
  CostStatisticsType,
} from './types'

const DEFAULT_PAGE_SIZE = 20
const EXPORT_RECORDS_STORAGE_KEY = 'newapi:cost-statistics:export-records'
const XLSX_MIME =
  'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet'

const DEFAULT_DATA: CostStatisticsData = {
  page: 1,
  page_size: DEFAULT_PAGE_SIZE,
  total: 0,
  items: [],
  summary: {
    quota: 0,
    request_count: 0,
    prompt_tokens: 0,
    completion_tokens: 0,
    total_tokens: 0,
  },
  period: 'today',
  start_timestamp: 0,
  end_timestamp: 0,
}

const PERIOD_OPTIONS: Array<{ value: CostStatisticsPeriod; label: string }> = [
  { value: 'today', label: 'Today' },
  { value: 'yesterday', label: 'Yesterday' },
  { value: 'week', label: 'This Week Range' },
  { value: 'month', label: 'This Month Range' },
  { value: 'last_month', label: 'Last Month Range' },
]

const TYPE_OPTIONS: Array<{ value: CostStatisticsType; label: string }> = [
  { value: 'all', label: 'All Types' },
  { value: 'consume', label: 'Consumption' },
  { value: 'refund', label: 'Refund' },
]

type CostStatisticsColumnId =
  | 'date'
  | 'user'
  | 'token'
  | 'model'
  | 'tokens'
  | 'requests'
  | 'duration'
  | 'groupRatio'
  | 'cost'

type CostStatisticsColumnVisibility = Record<CostStatisticsColumnId, boolean>

type ExportRecord = {
  id: string
  filename: string
  createdAt: number
  size: number
  rowCount: number
  content: string
}

type XlsxCell = string | number | null

const DEFAULT_COLUMN_VISIBILITY: CostStatisticsColumnVisibility = {
  date: true,
  user: true,
  token: true,
  model: true,
  tokens: true,
  requests: true,
  duration: false,
  groupRatio: true,
  cost: true,
}

const COLUMN_TOGGLE_OPTIONS: Array<{
  id: CostStatisticsColumnId
  label: string
}> = [
  { id: 'date', label: 'Date' },
  { id: 'token', label: 'Token' },
  { id: 'model', label: 'Model' },
  { id: 'tokens', label: 'Tokens' },
  { id: 'requests', label: 'Requests' },
  { id: 'duration', label: 'Duration' },
  { id: 'groupRatio', label: 'Group Ratio' },
  { id: 'cost', label: 'Cost' },
]

function defaultFilters(): CostStatisticsFilters {
  return {
    period: 'today',
    type: 'all',
    model: '',
    token: '',
    group: '',
    username: '',
    channel: '',
    start_timestamp: undefined,
    end_timestamp: undefined,
  }
}

function getLogTypeLabel(t: (key: string) => string, type: number) {
  if (type === 6) return t('Refund')
  return t('Consumption')
}

function formatGroupRatio(value: number) {
  if (!value) return '-'
  return `${value.toFixed(2)}x`
}

function formatDuration(seconds: number) {
  if (!seconds) return '-'
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  const restSeconds = seconds % 60
  if (minutes < 60)
    return restSeconds ? `${minutes}m ${restSeconds}s` : `${minutes}m`
  const hours = Math.floor(minutes / 60)
  const restMinutes = minutes % 60
  return restMinutes ? `${hours}h ${restMinutes}m` : `${hours}h`
}

function formatRangeLabel(start: number, end: number) {
  if (!start || !end) return '-'
  return `${formatTimestampToDate(start).slice(0, 16)} ~ ${formatTimestampToDate(end).slice(0, 16)}`
}

function formatExportDate(timestamp: number) {
  const date = new Date(timestamp)
  const pad = (value: number) => String(value).padStart(2, '0')
  return `${date.getFullYear()}/${pad(date.getMonth() + 1)}/${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

function formatExportSize(size: number) {
  if (size < 1024) return `${size} B`
  return `${Math.max(1, Math.round(size / 1024))} KB`
}

function getPeriodRange(period: CostStatisticsPeriod) {
  const now = new Date()
  const start = new Date(now)
  start.setHours(0, 0, 0, 0)
  let end = now

  if (period === 'week') {
    const offset = (now.getDay() + 6) % 7
    start.setDate(now.getDate() - offset)
  } else if (period === 'yesterday') {
    start.setDate(now.getDate() - 1)
    end = new Date(now)
    end.setHours(0, 0, 0, 0)
    end = new Date(end.getTime() - 1000)
  } else if (period === 'month') {
    start.setDate(1)
  } else if (period === 'last_month') {
    start.setDate(1)
    start.setMonth(now.getMonth() - 1)
    end = new Date(now.getFullYear(), now.getMonth(), 1, 0, 0, 0, 0)
    end = new Date(end.getTime() - 1000)
  }

  return {
    start: Math.floor(start.getTime() / 1000),
    end: Math.floor(end.getTime() / 1000),
  }
}

function getFilterRange(filters: CostStatisticsFilters) {
  if (
    filters.period === 'custom' &&
    filters.start_timestamp &&
    filters.end_timestamp
  ) {
    return {
      start: filters.start_timestamp,
      end: filters.end_timestamp,
    }
  }
  return getPeriodRange(filters.period)
}

function HeaderLabel(props: { children: React.ReactNode; align?: 'right' }) {
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1',
        props.align === 'right' && 'justify-end'
      )}
    >
      {props.children}
      <ChevronsUpDown className='text-muted-foreground size-3.5' />
    </span>
  )
}

function escapeXmlCell(value: string | number) {
  return String(value ?? '')
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;')
}

function downloadTextFile(filename: string, content: string, type: string) {
  const blob = new Blob([content], { type })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}

function downloadBinaryFile(
  filename: string,
  content: Uint8Array,
  type: string
) {
  const buffer = new ArrayBuffer(content.byteLength)
  new Uint8Array(buffer).set(content)
  const blob = new Blob([buffer], { type })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}

function downloadExportRecord(record: ExportRecord) {
  if (record.content.trimStart().startsWith('<')) {
    downloadTextFile(
      record.filename,
      record.content,
      'application/vnd.ms-excel;charset=utf-8'
    )
    return
  }
  downloadBinaryFile(record.filename, base64ToBytes(record.content), XLSX_MIME)
}

function bytesToBase64(bytes: Uint8Array) {
  let binary = ''
  const chunkSize = 0x8000
  for (let index = 0; index < bytes.length; index += chunkSize) {
    const chunk = bytes.subarray(index, index + chunkSize)
    binary += String.fromCharCode(...chunk)
  }
  return window.btoa(binary)
}

function base64ToBytes(base64: string) {
  const binary = window.atob(base64)
  const bytes = new Uint8Array(binary.length)
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index)
  }
  return bytes
}

const COST_EXPORT_HEADERS = [
  '日期',
  '类型',
  '模型',
  '令牌',
  'API Key（脱敏）',
  '分组',
  '用户',
  '请求数',
  '计费模式',
  '匹配阶梯',
  '提示词 Token',
  '输入 Token（不含缓存）',
  '提示词单价 ($/1M)',
  '补全 Token',
  '补全单价 ($/1M)',
  '缓存读取 Token',
  '缓存读取单价 ($/1M)',
  '缓存写入（通用/5m）Token',
  '缓存写入（通用/5m）单价 ($/1M)',
  '缓存写入1h Token',
  '缓存写入1h单价 ($/1M)',
  '固定单价 ($/次)',
  '分组倍率',
  '费用 ($)',
  '原价 ($)',
] as const

function formatExportDateValue(date: string) {
  if (/^\d{8}$/.test(date)) {
    return `${date.slice(0, 4)}-${date.slice(4, 6)}-${date.slice(6, 8)}`
  }
  return date
}

function getExportTypeLabel(type: number) {
  return type === 6 ? '退款' : '消耗'
}

function nullableNumber(value: number | null | undefined) {
  if (value == null || !Number.isFinite(value) || value === 0) return null
  return value
}

function nullablePositiveNumber(value: number | null | undefined) {
  if (value == null || !Number.isFinite(value) || value <= 0) return null
  return value
}

function getCostUsd(row: CostStatisticsRow) {
  return quotaUnitsToDollars(row.quota || 0)
}

function getOriginalCostUsd(row: CostStatisticsRow) {
  const cost = getCostUsd(row)
  if (!row.group_ratio) return cost
  return cost / row.group_ratio
}

function getCostExportRows(rows: CostStatisticsRow[]): XlsxCell[][] {
  return rows.map((row) => [
    formatExportDateValue(row.date),
    getExportTypeLabel(row.type),
    row.model_name || '',
    row.token_name || '',
    row.token_key_masked || '',
    row.group || '',
    row.username || '',
    row.request_count || 0,
    row.billing_mode || (row.model_price > 0 ? '按次计费' : '按量倍率'),
    row.matched_tier || '',
    row.prompt_tokens || 0,
    row.input_tokens || 0,
    nullableNumber(row.input_unit_price),
    row.completion_tokens || 0,
    nullableNumber(row.output_unit_price),
    row.cache_tokens || 0,
    nullableNumber(row.cache_read_unit_price),
    row.cache_write_5m_tokens || 0,
    nullableNumber(row.cache_write_unit_price),
    row.cache_write_1h_tokens || 0,
    nullableNumber(row.cache_write_1h_unit_price),
    nullablePositiveNumber(row.model_price),
    nullableNumber(row.group_ratio),
    getCostUsd(row),
    getOriginalCostUsd(row),
  ])
}

function formatCostExportDateTime(timestamp: number) {
  return formatTimestampToDate(timestamp).slice(0, 19)
}

function formatExportDay(timestamp: number) {
  const formatted = formatTimestampToDate(timestamp).slice(0, 10)
  if (formatted === '-') return ''
  return formatted.replaceAll('-', '')
}

function getCostExportInfoRows(
  data: CostStatisticsData,
  filters: CostStatisticsFilters,
  subjectLabel?: string
): XlsxCell[][] {
  return [
    ['费用导出说明'],
    ['导出时间', formatCostExportDateTime(Math.floor(Date.now() / 1000))],
    [
      '时间范围',
      `${formatExportDay(data.start_timestamp)} ~ ${formatExportDay(data.end_timestamp)}`,
    ],
    ['用户', subjectLabel || filters.username || '全部'],
    ['类型', getCostExportFilterTypeLabel(filters.type)],
    ['记录数', String(data.total)],
    [''],
    ['字段与计算口径'],
    ['费用 ($)', '实际扣费金额（最终价格），退款记录按负数统计'],
    ['原价 ($)', '费用 ÷ 分组倍率（即未享分组折扣前的金额）'],
    ['分组倍率', '分组折扣倍率，小于 1 表示折扣'],
    ['缓存 Token', '缓存读取 + 缓存写入（通用/5m）+ 缓存写入1h'],
    [
      '输入 Token（不含缓存）',
      '按原价计费的输入 Token；Claude 类模型的“提示词 Token”本身已不含缓存，其余模型为提示词 Token 减去缓存读取与缓存写入；汇总（按小时/按天）导出中，本列在功能上线前的历史周期显示为 0，需管理员对该时间范围触发重算后才会补全',
    ],
    ['计费模式', '按量倍率 / 按次计费 / 动态计费'],
    [
      'Token 单价',
      '统一导出实际 $ / 1M 单价：按量倍率由模型倍率换算，动态计费按每条日志保存的表达式与匹配阶梯解析',
    ],
    ['类型枚举', '消耗 / 退款'],
    ['单价单位', '$ / 1M tokens 或 $ / 次'],
    [
      '统计延迟',
      '统计数据每 20 分钟按已完成小时聚合一次，今日数据最长延迟约 1.5 小时',
    ],
  ]
}

function getCostExportFilterTypeLabel(type: CostStatisticsType) {
  switch (type) {
    case 'consume':
      return '消耗'
    case 'refund':
      return '退款'
    default:
      return '全部'
  }
}

function getColumnName(index: number) {
  let value = index + 1
  let name = ''
  while (value > 0) {
    const remainder = (value - 1) % 26
    name = String.fromCharCode(65 + remainder) + name
    value = Math.floor((value - 1) / 26)
  }
  return name
}

function getColumnWidth(rows: XlsxCell[][], index: number) {
  const maxLength = rows.reduce((max, row) => {
    return Math.max(max, String(row[index] ?? '').length)
  }, 8)
  return Math.min(Math.max(maxLength + 2, 10), 54)
}

function getCellStyle(rowIndex: number, columnIndex: number, value: XlsxCell) {
  if (rowIndex === 0) return 1
  if (typeof value !== 'number') return 2
  if ([6, 9, 10, 12, 14, 16, 18].includes(columnIndex)) return 3
  if (columnIndex === 21) return 5
  return 4
}

function buildCellXml(cellRef: string, value: XlsxCell, style: number) {
  if (value == null || value === '') {
    return `<c r="${cellRef}" s="${style}"></c>`
  }
  if (typeof value === 'number') {
    return `<c r="${cellRef}" s="${style}"><v>${value}</v></c>`
  }
  return `<c r="${cellRef}" s="${style}" t="inlineStr"><is><t>${escapeXmlCell(value)}</t></is></c>`
}

function buildWorksheetXml(rows: XlsxCell[][]) {
  const columnCount = rows.reduce((max, row) => Math.max(max, row.length), 1)
  const colsXml = Array.from({ length: columnCount })
    .map((_, index) => {
      const width = getColumnWidth(rows, index)
      return `<col min="${index + 1}" max="${index + 1}" width="${width}" customWidth="1"/>`
    })
    .join('')
  const rowsXml = rows
    .map((row, rowIndex) => {
      const rowNumber = rowIndex + 1
      const cellsXml = row
        .map((cell, columnIndex) => {
          const cellRef = `${getColumnName(columnIndex)}${rowNumber}`
          return buildCellXml(
            cellRef,
            cell,
            getCellStyle(rowIndex, columnIndex, cell)
          )
        })
        .join('')
      return `<row r="${rowNumber}">${cellsXml}</row>`
    })
    .join('')

  return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <dimension ref="A1:${getColumnName(columnCount - 1)}${rows.length}"/>
  <sheetViews>
    <sheetView workbookViewId="0" showGridLines="1"/>
  </sheetViews>
  <sheetFormatPr defaultRowHeight="15"/>
  <cols>${colsXml}</cols>
  <sheetData>${rowsXml}</sheetData>
</worksheet>`
}

function buildXlsxContent(
  rows: CostStatisticsRow[],
  data: CostStatisticsData,
  filters: CostStatisticsFilters,
  subjectLabel?: string
) {
  const sheetRows: XlsxCell[][] = [
    Array.from(COST_EXPORT_HEADERS),
    ...getCostExportRows(rows),
  ]
  const infoRows = getCostExportInfoRows(data, filters, subjectLabel)
  return createZip([
    {
      path: '[Content_Types].xml',
      content: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/>
  <Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
  <Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>
  <Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
  <Override PartName="/xl/worksheets/sheet2.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>
</Types>`,
    },
    {
      path: '_rels/.rels',
      content: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/>
</Relationships>`,
    },
    {
      path: 'docProps/app.xml',
      content: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties" xmlns:vt="http://schemas.openxmlformats.org/officeDocument/2006/docPropsVTypes">
  <Application>New API</Application>
</Properties>`,
    },
    {
      path: 'docProps/core.xml',
      content: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <dc:creator>New API</dc:creator>
  <dcterms:created xsi:type="dcterms:W3CDTF">${new Date().toISOString()}</dcterms:created>
</cp:coreProperties>`,
    },
    {
      path: 'xl/workbook.xml',
      content: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <sheets>
    <sheet name="数据" sheetId="1" r:id="rId1"/>
    <sheet name="说明" sheetId="2" r:id="rId2"/>
  </sheets>
</workbook>`,
    },
    {
      path: 'xl/_rels/workbook.xml.rels',
      content: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/>
  <Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet2.xml"/>
  <Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>
</Relationships>`,
    },
    {
      path: 'xl/styles.xml',
      content: `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <numFmts count="3">
    <numFmt numFmtId="164" formatCode="#,##0"/>
    <numFmt numFmtId="165" formatCode="0.000000"/>
    <numFmt numFmtId="166" formatCode="0.00"/>
  </numFmts>
  <fonts count="2">
    <font><sz val="11"/><name val="Calibri"/><family val="2"/></font>
    <font><b/><sz val="11"/><name val="Calibri"/><family val="2"/></font>
  </fonts>
  <fills count="2">
    <fill><patternFill patternType="none"/></fill>
    <fill><patternFill patternType="gray125"/></fill>
  </fills>
  <borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders>
  <cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>
  <cellXfs count="7">
    <xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>
    <xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0" applyFont="true" applyAlignment="true"><alignment horizontal="left"/></xf>
    <xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0" applyAlignment="true"><alignment horizontal="left"/></xf>
    <xf numFmtId="164" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="true" applyAlignment="true"><alignment horizontal="left"/></xf>
    <xf numFmtId="165" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="true" applyAlignment="true"><alignment horizontal="left"/></xf>
    <xf numFmtId="166" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="true" applyAlignment="true"><alignment horizontal="left"/></xf>
    <xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0" applyFont="true"/>
  </cellXfs>
  <cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles>
</styleSheet>`,
    },
    {
      path: 'xl/worksheets/sheet1.xml',
      content: buildWorksheetXml(sheetRows),
    },
    {
      path: 'xl/worksheets/sheet2.xml',
      content: buildWorksheetXml(infoRows),
    },
  ])
}

type ZipFile = {
  path: string
  content: string
}

const CRC32_TABLE = (() => {
  const table = new Uint32Array(256)
  for (let index = 0; index < 256; index += 1) {
    let value = index
    for (let bit = 0; bit < 8; bit += 1) {
      value = value & 1 ? 0xedb88320 ^ (value >>> 1) : value >>> 1
    }
    table[index] = value >>> 0
  }
  return table
})()

function getCrc32(bytes: Uint8Array) {
  let value = 0xffffffff
  for (const byte of bytes) {
    value = CRC32_TABLE[(value ^ byte) & 0xff] ^ (value >>> 8)
  }
  return (value ^ 0xffffffff) >>> 0
}

function getDosDateTime(date: Date) {
  const year = Math.max(date.getFullYear(), 1980)
  return {
    date: ((year - 1980) << 9) | ((date.getMonth() + 1) << 5) | date.getDate(),
    time:
      (date.getHours() << 11) |
      (date.getMinutes() << 5) |
      Math.floor(date.getSeconds() / 2),
  }
}

function concatBytes(parts: Uint8Array[], totalLength: number) {
  const result = new Uint8Array(totalLength)
  let offset = 0
  for (const part of parts) {
    result.set(part, offset)
    offset += part.length
  }
  return result
}

function createZip(files: ZipFile[]) {
  const encoder = new TextEncoder()
  const now = getDosDateTime(new Date())
  const localParts: Uint8Array[] = []
  const centralParts: Uint8Array[] = []
  let localOffset = 0
  let localTotalLength = 0
  let centralTotalLength = 0

  for (const file of files) {
    const nameBytes = encoder.encode(file.path)
    const contentBytes = encoder.encode(file.content)
    const crc = getCrc32(contentBytes)

    const localHeader = new Uint8Array(30 + nameBytes.length)
    const localView = new DataView(localHeader.buffer)
    localView.setUint32(0, 0x04034b50, true)
    localView.setUint16(4, 20, true)
    localView.setUint16(6, 0x0800, true)
    localView.setUint16(8, 0, true)
    localView.setUint16(10, now.time, true)
    localView.setUint16(12, now.date, true)
    localView.setUint32(14, crc, true)
    localView.setUint32(18, contentBytes.length, true)
    localView.setUint32(22, contentBytes.length, true)
    localView.setUint16(26, nameBytes.length, true)
    localView.setUint16(28, 0, true)
    localHeader.set(nameBytes, 30)
    localParts.push(localHeader, contentBytes)
    localTotalLength += localHeader.length + contentBytes.length

    const centralHeader = new Uint8Array(46 + nameBytes.length)
    const centralView = new DataView(centralHeader.buffer)
    centralView.setUint32(0, 0x02014b50, true)
    centralView.setUint16(4, 20, true)
    centralView.setUint16(6, 20, true)
    centralView.setUint16(8, 0x0800, true)
    centralView.setUint16(10, 0, true)
    centralView.setUint16(12, now.time, true)
    centralView.setUint16(14, now.date, true)
    centralView.setUint32(16, crc, true)
    centralView.setUint32(20, contentBytes.length, true)
    centralView.setUint32(24, contentBytes.length, true)
    centralView.setUint16(28, nameBytes.length, true)
    centralView.setUint16(30, 0, true)
    centralView.setUint16(32, 0, true)
    centralView.setUint16(34, 0, true)
    centralView.setUint16(36, 0, true)
    centralView.setUint32(38, 0, true)
    centralView.setUint32(42, localOffset, true)
    centralHeader.set(nameBytes, 46)
    centralParts.push(centralHeader)
    centralTotalLength += centralHeader.length

    localOffset += localHeader.length + contentBytes.length
  }

  const endRecord = new Uint8Array(22)
  const endView = new DataView(endRecord.buffer)
  endView.setUint32(0, 0x06054b50, true)
  endView.setUint16(4, 0, true)
  endView.setUint16(6, 0, true)
  endView.setUint16(8, files.length, true)
  endView.setUint16(10, files.length, true)
  endView.setUint32(12, centralTotalLength, true)
  endView.setUint32(16, localTotalLength, true)
  endView.setUint16(20, 0, true)

  return concatBytes(
    [...localParts, ...centralParts, endRecord],
    localTotalLength + centralTotalLength + endRecord.length
  )
}

function buildExportFilename() {
  const date = new Date()
  const pad = (value: number) => String(value).padStart(2, '0')
  return `cost_aggregate_${date.getFullYear()}${pad(date.getMonth() + 1)}${pad(date.getDate())}_${pad(date.getHours())}${pad(date.getMinutes())}${pad(date.getSeconds())}.xlsx`
}

function loadStoredExportRecords(): ExportRecord[] {
  try {
    const raw = window.localStorage.getItem(EXPORT_RECORDS_STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    return Array.isArray(parsed) ? parsed.slice(0, 20) : []
  } catch {
    return []
  }
}

function storeExportRecords(records: ExportRecord[]) {
  window.localStorage.setItem(
    EXPORT_RECORDS_STORAGE_KEY,
    JSON.stringify(records.slice(0, 20))
  )
}

function SummaryBadge(props: {
  label: string
  value: string
  loading: boolean
}) {
  return (
    <Badge
      variant='outline'
      className='bg-background h-7 rounded-full px-2.5 font-normal shadow-none'
    >
      <span className='text-muted-foreground'>{props.label}</span>
      <span className='font-mono font-semibold tabular-nums'>
        {props.loading ? <Skeleton className='h-3.5 w-12' /> : props.value}
      </span>
    </Badge>
  )
}

function ColumnVisibilityMenu(props: {
  visibility: CostStatisticsColumnVisibility
  onVisibilityChange: (visibility: CostStatisticsColumnVisibility) => void
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)

  const toggleColumn = (id: CostStatisticsColumnId, checked: boolean) => {
    props.onVisibilityChange({
      ...props.visibility,
      [id]: checked,
    })
  }

  return (
    <DropdownMenu modal={false} open={open} onOpenChange={setOpen}>
      <DropdownMenuTrigger
        render={
          <Button
            variant='outline'
            size='sm'
            className='rounded-full'
            onMouseEnter={() => setOpen(true)}
            onFocus={() => setOpen(true)}
          />
        }
      >
        <Eye data-icon='inline-start' />
        {t('View')}
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align='end'
        className='w-40'
        onMouseEnter={() => setOpen(true)}
        onMouseLeave={() => setOpen(false)}
      >
        <DropdownMenuGroup>
          <DropdownMenuLabel>{t('Toggle columns')}</DropdownMenuLabel>
          {COLUMN_TOGGLE_OPTIONS.map((column) => (
            <DropdownMenuCheckboxItem
              key={column.id}
              checked={props.visibility[column.id]}
              onCheckedChange={(value) => toggleColumn(column.id, !!value)}
            >
              {t(column.label)}
            </DropdownMenuCheckboxItem>
          ))}
        </DropdownMenuGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

function ExportRecordsPopover(props: {
  records: ExportRecord[]
  onDownload: (record: ExportRecord) => void
  onDelete: (id: string) => void
  onClear: () => void
}) {
  const { t } = useTranslation()

  return (
    <Popover>
      <PopoverTrigger
        render={<Button variant='outline' size='sm' className='rounded-full' />}
      >
        <Clock3 data-icon='inline-start' />
        {t('Export Records')}
      </PopoverTrigger>
      <PopoverContent
        align='end'
        className='w-[min(440px,calc(100vw-2rem))] p-0'
      >
        <div className='flex items-center justify-between border-b px-4 py-3'>
          <div className='font-medium'>{t('Export Records')}</div>
          {props.records.length > 0 && (
            <Button
              variant='ghost'
              size='sm'
              className='h-7 rounded-full text-xs'
              onClick={props.onClear}
            >
              <Trash2 data-icon='inline-start' />
              {t('Clear all')}
            </Button>
          )}
        </div>
        {props.records.length === 0 ? (
          <div className='flex min-h-32 flex-col items-center justify-center gap-3 p-6 text-center'>
            <Clock3 className='text-muted-foreground size-9' />
            <div className='text-muted-foreground text-sm'>
              {t('No export records')}
            </div>
          </div>
        ) : (
          <div className='max-h-80 overflow-auto p-2'>
            {props.records.map((record) => (
              <div
                key={record.id}
                className='hover:bg-muted/50 flex items-center gap-3 rounded-lg p-2'
              >
                <div className='bg-primary/10 text-primary flex size-5 shrink-0 items-center justify-center rounded-full'>
                  <span className='text-xs'>✓</span>
                </div>
                <div className='min-w-0 flex-1'>
                  <div className='truncate text-sm font-medium'>
                    {record.filename}
                  </div>
                  <div className='text-muted-foreground text-xs'>
                    {t('Completed')} · {formatExportSize(record.size)}
                  </div>
                  <div className='text-muted-foreground text-xs'>
                    {formatExportDate(record.createdAt)}
                  </div>
                </div>
                <Button
                  variant='ghost'
                  size='icon'
                  className='rounded-full'
                  onClick={() => props.onDownload(record)}
                >
                  <Download />
                </Button>
                <Button
                  variant='ghost'
                  size='icon'
                  className='rounded-full'
                  onClick={() => props.onDelete(record.id)}
                >
                  <Trash2 />
                </Button>
              </div>
            ))}
          </div>
        )}
      </PopoverContent>
    </Popover>
  )
}

function DateRangePicker(props: {
  period: CostStatisticsPeriod
  start: number
  end: number
  maxCustomRangeSeconds?: number
  onPeriodChange: (period: CostStatisticsPeriod) => void
  onRangeChange: (start: number, end: number) => void
}) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [draftStart, setDraftStart] = useState('')
  const [draftEnd, setDraftEnd] = useState('')
  const activeLabel =
    props.period === 'custom'
      ? 'Custom Range'
      : PERIOD_OPTIONS.find((option) => option.value === props.period)?.label ||
        'Today'

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) {
      setDraftStart(formatTimestampForInput(props.start))
      setDraftEnd(formatTimestampForInput(props.end))
    }
    setOpen(nextOpen)
  }

  const handlePeriodChange = (period: CostStatisticsPeriod) => {
    props.onPeriodChange(period)
    setOpen(false)
  }

  const applyCustomRange = () => {
    const start = parseTimestampFromInput(draftStart)
    const end = parseTimestampFromInput(draftEnd)
    const validationError = validateCostStatisticsCustomRange(
      start,
      end,
      props.maxCustomRangeSeconds
    )
    if (
      validationError === 'invalid_time' ||
      validationError === 'end_before_start'
    ) {
      toast.error(t('End time must be later than start time'))
      return
    }
    if (validationError === 'range_too_long') {
      toast.error(t('Custom range cannot exceed 31 days'))
      return
    }
    props.onRangeChange(start, end)
    setOpen(false)
  }

  return (
    <Popover open={open} onOpenChange={handleOpenChange}>
      <PopoverTrigger
        render={
          <Button
            type='button'
            variant='outline'
            className='h-9 min-w-0 flex-1 justify-start rounded-full px-3 font-normal tabular-nums lg:min-w-[480px]'
          />
        }
      >
        <CalendarDays data-icon='inline-start' />
        <span className='truncate'>
          {formatRangeLabel(props.start, props.end)}
        </span>
      </PopoverTrigger>
      <PopoverContent align='start' className='w-[min(420px,calc(100vw-2rem))]'>
        <div className='flex flex-col gap-3'>
          <div className='grid gap-2 sm:grid-cols-[1fr_auto_1fr] sm:items-end'>
            <div className='flex flex-col gap-1.5'>
              <div className='text-muted-foreground text-xs'>
                {t('Start Time')}
              </div>
              <Input
                type='datetime-local'
                value={draftStart}
                onChange={(event) => setDraftStart(event.target.value)}
                className='h-8 font-mono text-xs tabular-nums'
              />
            </div>
            <span className='text-muted-foreground hidden pb-2 text-xs sm:block'>
              ~
            </span>
            <div className='flex flex-col gap-1.5'>
              <div className='text-muted-foreground text-xs'>
                {t('End Time')}
              </div>
              <Input
                type='datetime-local'
                value={draftEnd}
                onChange={(event) => setDraftEnd(event.target.value)}
                className='h-8 font-mono text-xs tabular-nums'
              />
            </div>
          </div>
          <div className='grid grid-cols-2 gap-2 sm:grid-cols-5'>
            {PERIOD_OPTIONS.map((option) => (
              <Button
                key={option.value}
                type='button'
                variant={
                  props.period === option.value ? 'default' : 'secondary'
                }
                size='sm'
                className='h-8'
                onClick={() => handlePeriodChange(option.value)}
              >
                {t(option.label)}
              </Button>
            ))}
          </div>
          <Button
            type='button'
            variant={props.period === 'custom' ? 'default' : 'outline'}
            size='sm'
            className='h-8'
            onClick={applyCustomRange}
          >
            {t('Custom Range')}
          </Button>
          <div className='text-muted-foreground text-xs'>{t(activeLabel)}</div>
        </div>
      </PopoverContent>
    </Popover>
  )
}

function PaginationControls(props: {
  page: number
  pageSize: number
  total: number
  onPageChange: (page: number) => void
}) {
  const { t } = useTranslation()
  const totalPages = Math.max(1, Math.ceil(props.total / props.pageSize))
  const pages = Array.from({ length: Math.min(totalPages, 3) }, (_, index) => {
    const start = Math.max(1, Math.min(props.page - 1, totalPages - 2))
    return start + index
  }).filter((page) => page <= totalPages)

  return (
    <div className='flex flex-wrap items-center justify-end gap-2 text-sm'>
      <span className='text-muted-foreground'>
        {t('Total')}:{' '}
        <span className='text-foreground'>{formatNumber(props.total)}</span>
      </span>
      <span className='text-muted-foreground'>{t('Rows per page')}</span>
      <Select value={String(props.pageSize)} onValueChange={() => undefined}>
        <SelectTrigger size='sm' className='h-9 w-[76px] rounded-full'>
          <SelectValue>{props.pageSize}</SelectValue>
        </SelectTrigger>
        <SelectContent alignItemWithTrigger={false}>
          <SelectGroup>
            <SelectItem value={String(props.pageSize)}>
              {props.pageSize}
            </SelectItem>
          </SelectGroup>
        </SelectContent>
      </Select>
      <Button
        variant='ghost'
        size='icon'
        className='rounded-full'
        disabled={props.page <= 1}
        onClick={() => props.onPageChange(1)}
      >
        «
      </Button>
      <Button
        variant='ghost'
        size='icon'
        className='rounded-full'
        disabled={props.page <= 1}
        onClick={() => props.onPageChange(props.page - 1)}
      >
        ‹
      </Button>
      {pages.map((page) => (
        <Button
          key={page}
          variant={page === props.page ? 'default' : 'ghost'}
          size='icon'
          className='rounded-full font-mono tabular-nums'
          onClick={() => props.onPageChange(page)}
        >
          {page}
        </Button>
      ))}
      <Button
        variant='ghost'
        size='icon'
        className='rounded-full'
        disabled={props.page >= totalPages}
        onClick={() => props.onPageChange(props.page + 1)}
      >
        ›
      </Button>
      <Button
        variant='ghost'
        size='icon'
        className='rounded-full'
        disabled={props.page >= totalPages}
        onClick={() => props.onPageChange(totalPages)}
      >
        »
      </Button>
    </div>
  )
}

function CostStatisticsTable(props: {
  data: CostStatisticsRow[]
  loading: boolean
  columnVisibility: CostStatisticsColumnVisibility
  showUserColumn: boolean
}) {
  const { t } = useTranslation()
  const columns = useMemo(
    () =>
      [
        {
          id: 'date',
          header: <HeaderLabel>{t('Date')}</HeaderLabel>,
          className: 'w-[120px]',
          cell: (row: CostStatisticsRow) => (
            <div className='flex flex-col gap-1'>
              <span className='font-mono tabular-nums'>{row.date}</span>
              <span className='text-primary text-xs'>
                {getLogTypeLabel(t, row.type)}
              </span>
            </div>
          ),
        },
        ...(props.showUserColumn
          ? [
              {
                id: 'user',
                header: <HeaderLabel>{t('User')}</HeaderLabel>,
                className: 'min-w-[150px]',
                cell: (row: CostStatisticsRow) => (
                  <span className='block max-w-[180px] truncate'>
                    {row.username || '-'}
                  </span>
                ),
              },
            ]
          : []),
        {
          id: 'token',
          header: <HeaderLabel>{t('Token')}</HeaderLabel>,
          className: 'min-w-[180px]',
          cell: (row: CostStatisticsRow) => (
            <div className='flex min-w-0 flex-col gap-1'>
              <div>
                <Badge
                  variant='outline'
                  className='bg-background max-w-full rounded-full font-medium'
                >
                  <KeyRound className='size-3.5' />
                  <span className='truncate'>{row.token_name || '-'}</span>
                </Badge>
              </div>
              <span className='text-muted-foreground truncate text-xs'>
                {row.group || (row.token_id > 0 ? `#${row.token_id}` : '-')}
              </span>
            </div>
          ),
        },
        {
          id: 'model',
          header: <HeaderLabel>{t('Model')}</HeaderLabel>,
          className: 'min-w-[220px]',
          cell: (row: CostStatisticsRow) =>
            row.model_name ? <ModelBadge modelName={row.model_name} /> : '-',
        },
        {
          id: 'tokens',
          header: <HeaderLabel>{t('Tokens')}</HeaderLabel>,
          className: 'w-[180px]',
          cell: (row: CostStatisticsRow) => (
            <div className='flex flex-col gap-1 font-mono tabular-nums'>
              <span
                title={`${formatNumber(row.prompt_tokens)} / ${formatNumber(row.completion_tokens)}`}
              >
                {formatTokens(row.prompt_tokens)} /{' '}
                {formatTokens(row.completion_tokens)}
              </span>
              {row.total_tokens > 0 && (
                <span
                  className='text-muted-foreground text-xs'
                  title={formatNumber(row.total_tokens)}
                >
                  {formatTokens(row.total_tokens)}
                </span>
              )}
            </div>
          ),
        },
        {
          id: 'requests',
          header: <HeaderLabel>{t('Requests')}</HeaderLabel>,
          className: 'w-[120px]',
          cell: (row: CostStatisticsRow) => formatNumber(row.request_count),
        },
        {
          id: 'duration',
          header: <HeaderLabel>{t('Duration')}</HeaderLabel>,
          className: 'w-[120px]',
          cell: (row: CostStatisticsRow) =>
            formatDuration(row.use_time_seconds),
        },
        {
          id: 'groupRatio',
          header: <HeaderLabel>{t('Group Ratio')}</HeaderLabel>,
          className: 'w-[150px]',
          cell: (row: CostStatisticsRow) => (
            <Badge
              variant='outline'
              className='bg-background rounded-full font-mono'
            >
              {formatGroupRatio(row.group_ratio)}
            </Badge>
          ),
        },
        {
          id: 'cost',
          header: <HeaderLabel align='right'>{t('Deduction')}</HeaderLabel>,
          className: 'w-[160px] text-right',
          cellClassName: 'text-right',
          cell: (row: CostStatisticsRow) => (
            <Badge
              variant='secondary'
              className='rounded-full font-mono font-semibold tabular-nums'
            >
              {formatLogQuota(row.quota)}
            </Badge>
          ),
        },
      ].filter(
        (column) => props.columnVisibility[column.id as CostStatisticsColumnId]
      ),
    [props.columnVisibility, props.showUserColumn, t]
  )

  if (props.loading) {
    return (
      <div className='flex flex-col gap-2 p-3'>
        {Array.from({ length: 8 }).map((_, index) => (
          <Skeleton key={index} className='h-12 w-full' />
        ))}
      </div>
    )
  }

  return (
    <StaticDataTable
      data={props.data}
      columns={columns}
      className='overflow-visible rounded-none border-0 [&_[data-slot=table-container]]:overflow-visible'
      tableClassName='min-w-[980px] [&_th]:sticky [&_th]:top-0 [&_th]:z-20 [&_th]:bg-card [&_th]:shadow-[0_1px_0_var(--border)]'
      headerRowClassName='bg-card hover:bg-card'
      getRowKey={(row) =>
        [
          row.date,
          row.type,
          row.token_id,
          row.token_name,
          row.model_name,
          row.group,
          row.username,
          row.channel_id,
        ].join(':')
      }
      getRowClassName={() => 'h-[66px]'}
      emptyContent={
        <Empty className='border-0'>
          <EmptyHeader>
            <EmptyTitle>{t('No Cost Statistics Found')}</EmptyTitle>
            <EmptyDescription>
              {t('No cost statistics match the current filters.')}
            </EmptyDescription>
          </EmptyHeader>
        </Empty>
      }
    />
  )
}

export function CostStatistics() {
  const { t } = useTranslation()
  const isAdmin = useIsAdmin()
  const userRole = useAuthStore((state) => state.auth.user?.role)
  const isSales = userRole === ROLE.SALES
  const [page, setPage] = useState(1)
  const [searchVersion, setSearchVersion] = useState(0)
  const [expanded, setExpanded] = useState(false)
  const [draftFilters, setDraftFilters] = useState(defaultFilters)
  const [filters, setFilters] = useState(defaultFilters)
  const [exporting, setExporting] = useState(false)
  const [columnVisibility, setColumnVisibility] =
    useState<CostStatisticsColumnVisibility>(DEFAULT_COLUMN_VISIBILITY)
  const [exportRecords, setExportRecords] = useState<ExportRecord[]>(
    loadStoredExportRecords
  )
  const [subject, setSubject] = useState<CostStatisticsSubject>('self')

  const managedUsersQuery = useQuery({
    queryKey: ['managed-log-users'],
    queryFn: getManagedLogUsers,
    enabled: isSales,
    refetchOnReconnect: false,
    refetchOnWindowFocus: false,
  })

  const updateExportRecords = (records: ExportRecord[]) => {
    setExportRecords(records)
    storeExportRecords(records)
  }

  const { data, isLoading, isFetching, isPlaceholderData, dataUpdatedAt } =
    useQuery({
      queryKey: [
        'cost-statistics',
        isAdmin,
        subject,
        page,
        filters,
        searchVersion,
        t,
      ],
      queryFn: async () => {
        const result = await getCostStatistics({
          isAdmin,
          page,
          pageSize: DEFAULT_PAGE_SIZE,
          filters,
          subject,
        })
        if (!result.success) {
          toast.error(result.message || t('Failed to load cost statistics'))
          return DEFAULT_DATA
        }
        return result.data || DEFAULT_DATA
      },
      placeholderData: (previousData) => previousData,
      refetchOnMount: true,
      refetchOnReconnect: false,
      refetchOnWindowFocus: false,
      gcTime: 2 * 60 * 1000,
      staleTime: 2 * 60 * 1000,
    })

  const stats = data || DEFAULT_DATA
  const loadingContent = isLoading || isPlaceholderData
  const draftRange = getFilterRange(draftFilters)
  const managedUsers = managedUsersQuery.data?.data ?? []
  const subjectOptions = [
    { value: 'self', label: t('My data') },
    { value: 'managed', label: t('All assigned customers') },
    ...managedUsers.map((user) => ({
      value: `customer:${user.id}`,
      label: user.display_name
        ? `${user.username} (${user.display_name})`
        : user.username,
      keywords: `${user.username} ${user.display_name ?? ''} ${user.id}`,
    })),
  ]

  const applyFilters = () => {
    setPage(1)
    setFilters({ ...draftFilters })
    setSearchVersion((value) => value + 1)
  }

  const resetFilters = () => {
    const next = defaultFilters()
    setPage(1)
    setDraftFilters(next)
    setFilters(next)
    if (isSales) setSubject('self')
    setSearchVersion((value) => value + 1)
  }

  const updateDraft = (
    key: keyof CostStatisticsFilters,
    value: string | CostStatisticsPeriod | CostStatisticsType
  ) => {
    setDraftFilters((current) => ({ ...current, [key]: value }))
  }

  const handlePeriodChange = (period: CostStatisticsPeriod) => {
    setDraftFilters((current) => ({
      ...current,
      period,
      start_timestamp: undefined,
      end_timestamp: undefined,
    }))
  }

  const handleCustomRangeChange = (start: number, end: number) => {
    setDraftFilters((current) => ({
      ...current,
      period: 'custom',
      start_timestamp: start,
      end_timestamp: end,
    }))
  }

  const handleExportXlsx = async () => {
    if (isFetching || isPlaceholderData) return
    if (stats.total === 0) {
      toast.warning(t('No data to export'))
      return
    }

    setExporting(true)
    try {
      let rows = stats.items
      if (stats.total > stats.items.length) {
        const result = await getCostStatistics({
          isAdmin,
          page: 1,
          pageSize: 0,
          filters,
          subject,
          exportAll: true,
        })
        if (!result.success) {
          toast.error(result.message || t('Failed to load cost statistics'))
          return
        }
        rows = result.data?.items || []
      }
      if (rows.length === 0) {
        toast.warning(t('No data to export'))
        return
      }

      const filename = buildExportFilename()
      const selectedSubjectLabel = isSales
        ? subjectOptions.find((option) => option.value === subject)?.label
        : undefined
      const content = buildXlsxContent(
        rows,
        stats,
        filters,
        selectedSubjectLabel
      )
      downloadBinaryFile(filename, content, XLSX_MIME)
      const record: ExportRecord = {
        id: `${Date.now()}-${Math.random().toString(36).slice(2)}`,
        filename,
        createdAt: Date.now(),
        size: content.length,
        rowCount: rows.length,
        content: bytesToBase64(content),
      }
      updateExportRecords([record, ...exportRecords])
      toast.success(t('Export completed'))
    } finally {
      setExporting(false)
    }
  }

  const activeTypeLabel =
    TYPE_OPTIONS.find((option) => option.value === draftFilters.type)?.label ||
    'All Types'

  return (
    <Main className='bg-muted/30 p-3 sm:p-4'>
      <div className='bg-card flex min-h-0 flex-1 flex-col rounded-3xl border p-4 shadow-xs sm:p-6'>
        <h1 className='text-2xl font-bold tracking-tight'>
          {t('Cost Statistics')}
        </h1>

        <div className='mt-5 rounded-2xl border p-3'>
          <div className='flex flex-col gap-3'>
            <div className='flex flex-col gap-2 lg:flex-row'>
              {isSales && (
                <SearchableSubjectSelect
                  value={subject}
                  options={subjectOptions}
                  onValueChange={(value) => {
                    setSubject(value as CostStatisticsSubject)
                    setPage(1)
                  }}
                  ariaLabel={t('View subject')}
                  className='h-9 min-w-0 flex-1 justify-between rounded-full px-3 font-normal lg:max-w-[320px]'
                />
              )}
              <DateRangePicker
                period={draftFilters.period}
                start={draftRange.start}
                end={draftRange.end}
                maxCustomRangeSeconds={
                  isAdmin ? undefined : MAX_NON_ADMIN_CUSTOM_RANGE_SECONDS
                }
                onPeriodChange={handlePeriodChange}
                onRangeChange={handleCustomRangeChange}
              />

              <Select value='day' onValueChange={() => undefined}>
                <SelectTrigger className='h-9 flex-1 rounded-full lg:max-w-[370px]'>
                  <SelectValue>{t('Daily Query')}</SelectValue>
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    <SelectItem value='day'>{t('Daily Query')}</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>

              <Select
                value={draftFilters.type}
                onValueChange={(value) => {
                  updateDraft('type', (value || 'all') as CostStatisticsType)
                }}
              >
                <SelectTrigger className='h-9 flex-1 rounded-full lg:max-w-[370px]'>
                  <SelectValue>{t(activeTypeLabel)}</SelectValue>
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    {TYPE_OPTIONS.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {t(option.label)}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>

              <Button
                type='button'
                variant='ghost'
                className='h-9 rounded-full'
                onClick={() => setExpanded((value) => !value)}
              >
                {t(expanded ? 'Collapse' : 'Expand')}
                <ChevronDown
                  data-icon='inline-end'
                  className={cn(expanded && 'rotate-180')}
                />
              </Button>
            </div>

            {expanded && (
              <div
                className={cn(
                  'grid gap-2',
                  isAdmin ? 'sm:grid-cols-2 lg:grid-cols-5' : 'sm:grid-cols-3'
                )}
              >
                <Input
                  placeholder={t('Model')}
                  value={draftFilters.model}
                  onChange={(event) => updateDraft('model', event.target.value)}
                  className='h-9 rounded-full'
                />
                <Input
                  placeholder={t('Token Name')}
                  value={draftFilters.token}
                  onChange={(event) => updateDraft('token', event.target.value)}
                  className='h-9 rounded-full'
                />
                <Input
                  placeholder={t('Group')}
                  value={draftFilters.group}
                  onChange={(event) => updateDraft('group', event.target.value)}
                  className='h-9 rounded-full'
                />
                {isAdmin && (
                  <>
                    <Input
                      placeholder={t('Username')}
                      value={draftFilters.username}
                      onChange={(event) =>
                        updateDraft('username', event.target.value)
                      }
                      className='h-9 rounded-full'
                    />
                    <Input
                      placeholder={t('Channel ID')}
                      value={draftFilters.channel}
                      onChange={(event) =>
                        updateDraft('channel', event.target.value)
                      }
                      className='h-9 rounded-full'
                    />
                  </>
                )}
              </div>
            )}

            <div className='flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between'>
              <div className='flex min-w-0 flex-col gap-2'>
                <div className='flex flex-wrap items-center gap-2'>
                  <SummaryBadge
                    label={t('Deduction')}
                    value={formatLogQuota(stats.summary.quota)}
                    loading={loadingContent}
                  />
                  <SummaryBadge
                    label={t('Requests')}
                    value={formatNumber(stats.summary.request_count)}
                    loading={loadingContent}
                  />
                  <DataFreshnessIndicator updatedAt={dataUpdatedAt} />
                </div>
              </div>

              <div className='flex flex-wrap justify-start gap-2 lg:justify-end'>
                <Button
                  variant='outline'
                  size='sm'
                  className='rounded-full'
                  disabled={exporting || isFetching || isPlaceholderData}
                  onClick={handleExportXlsx}
                >
                  <Download data-icon='inline-start' />
                  {t('Export XLSX')}
                </Button>
                <ExportRecordsPopover
                  records={exportRecords}
                  onDownload={downloadExportRecord}
                  onDelete={(id) =>
                    updateExportRecords(
                      exportRecords.filter((record) => record.id !== id)
                    )
                  }
                  onClear={() => updateExportRecords([])}
                />
                <Button
                  variant='outline'
                  size='sm'
                  className='rounded-full'
                  onClick={resetFilters}
                >
                  <RotateCcw data-icon='inline-start' />
                  {t('Reset')}
                </Button>
                <Button
                  size='sm'
                  className='rounded-full'
                  onClick={applyFilters}
                >
                  <Search data-icon='inline-start' />
                  {t('Search')}
                </Button>
                <ColumnVisibilityMenu
                  visibility={columnVisibility}
                  onVisibilityChange={setColumnVisibility}
                />
              </div>
            </div>
          </div>
        </div>

        <div className='mt-4 min-h-0 flex-1 overflow-auto rounded-2xl border'>
          <CostStatisticsTable
            data={stats.items}
            loading={loadingContent}
            columnVisibility={columnVisibility}
            showUserColumn={isSales && subject !== 'self'}
          />
        </div>

        <div className='mt-3'>
          <PaginationControls
            page={page}
            pageSize={stats.page_size || DEFAULT_PAGE_SIZE}
            total={stats.total}
            onPageChange={setPage}
          />
        </div>
      </div>
    </Main>
  )
}
