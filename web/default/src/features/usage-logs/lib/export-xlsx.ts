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
import { formatTimestampToDate } from '@/lib/format'
import type { UsageLog } from '../data/schema'
import type { LogOtherData } from '../types'

type XlsxCell = string | number | null
type QuotaFormatter = (quota: number) => string

const HEADERS = [
  '时间',
  '类型',
  '用户',
  '令牌',
  '模型',
  '分组',
  '请求 ID',
  '上游请求 ID',
  '输入 Token',
  '输出 Token',
  '总 Token',
  '缓存 Token',
  '耗时（秒）',
  '首 Token 耗时（秒）',
  '流式',
  '扣费',
  '计费模式',
  '匹配阶梯',
  '流状态',
  '详情',
] as const

const LOG_TYPE_LABELS: Record<number, string> = {
  0: '未知',
  1: '充值',
  2: '消耗',
  3: '管理',
  4: '系统',
  5: '错误',
  6: '退款',
  7: '登录',
}

function parseOther(value: string): LogOtherData | null {
  if (!value) return null
  try {
    return JSON.parse(value) as LogOtherData
  } catch {
    return null
  }
}

function getCacheTokens(other: LogOtherData | null): number {
  return (
    (other?.cache_tokens || 0) +
    (other?.cache_creation_tokens || 0) +
    (other?.cache_creation_tokens_5m || 0) +
    (other?.cache_creation_tokens_1h || 0)
  )
}

function getBillingMode(other: LogOtherData | null): string {
  if (other?.billing_mode === 'tiered_expr') return '动态计费'
  if ((other?.model_price || 0) > 0) return '按次计费'
  return '按量倍率'
}

function getStreamStatus(log: UsageLog, other: LogOtherData | null): string {
  if (!log.is_stream) return ''
  const status = other?.stream_status
  if (!status) return '完成'
  if (status.status === 'error') {
    return status.end_reason ? `错误：${status.end_reason}` : '错误'
  }
  return status.status || '完成'
}

function getRows(logs: UsageLog[], formatQuota: QuotaFormatter): XlsxCell[][] {
  return logs.map((log) => {
    const other = parseOther(log.other)
    return [
      formatTimestampToDate(log.created_at),
      LOG_TYPE_LABELS[log.type] || '未知',
      log.username,
      log.token_name,
      log.model_name,
      log.group,
      log.request_id,
      log.upstream_request_id,
      log.prompt_tokens,
      log.completion_tokens,
      log.prompt_tokens + log.completion_tokens,
      getCacheTokens(other),
      log.use_time,
      other?.frt || null,
      log.is_stream ? '是' : '否',
      formatQuota(log.quota),
      getBillingMode(other),
      other?.matched_tier || '',
      getStreamStatus(log, other),
      log.content,
    ]
  })
}

function escapeXml(value: XlsxCell): string {
  const safeValue = Array.from(String(value ?? ''))
    .filter((character) => {
      const code = character.charCodeAt(0)
      return !(
        code <= 8 ||
        code === 11 ||
        code === 12 ||
        (code >= 14 && code <= 31)
      )
    })
    .join('')
  return safeValue
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;')
}

function getColumnName(index: number): string {
  let value = index + 1
  let name = ''
  while (value > 0) {
    name = String.fromCharCode(65 + ((value - 1) % 26)) + name
    value = Math.floor((value - 1) / 26)
  }
  return name
}

function buildWorksheet(rows: XlsxCell[][]): string {
  const lastColumn = getColumnName(HEADERS.length - 1)
  const columnWidths = rows[0].map((_, index) => {
    const length = rows.reduce(
      (max, row) => Math.max(max, String(row[index] ?? '').length),
      8
    )
    return Math.min(Math.max(length + 2, 10), 48)
  })
  const rowsXml = rows
    .map((row, rowIndex) => {
      const cells = row
        .map((value, columnIndex) => {
          const reference = `${getColumnName(columnIndex)}${rowIndex + 1}`
          const style = rowIndex === 0 ? 1 : typeof value === 'number' ? 3 : 2
          if (value === null || value === '') {
            return `<c r="${reference}" s="${style}"></c>`
          }
          if (typeof value === 'number') {
            return `<c r="${reference}" s="${style}"><v>${value}</v></c>`
          }
          return `<c r="${reference}" s="${style}" t="inlineStr"><is><t>${escapeXml(value)}</t></is></c>`
        })
        .join('')
      return `<row r="${rowIndex + 1}">${cells}</row>`
    })
    .join('')

  return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <dimension ref="A1:${lastColumn}${rows.length}"/>
  <sheetViews><sheetView workbookViewId="0"><pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews>
  <cols>${columnWidths.map((width, index) => `<col min="${index + 1}" max="${index + 1}" width="${width}" customWidth="1"/>`).join('')}</cols>
  <sheetData>${rowsXml}</sheetData>
  <autoFilter ref="A1:${lastColumn}${rows.length}"/>
</worksheet>`
}

type ZipFile = { path: string; content: string }

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

function crc32(bytes: Uint8Array): number {
  let value = 0xffffffff
  for (const byte of bytes) {
    value = CRC32_TABLE[(value ^ byte) & 0xff] ^ (value >>> 8)
  }
  return (value ^ 0xffffffff) >>> 0
}

function getDosDateTime(date: Date): { date: number; time: number } {
  const year = Math.max(date.getFullYear(), 1980)
  return {
    date: ((year - 1980) << 9) | ((date.getMonth() + 1) << 5) | date.getDate(),
    time:
      (date.getHours() << 11) |
      (date.getMinutes() << 5) |
      Math.floor(date.getSeconds() / 2),
  }
}

function createZip(files: ZipFile[]): Uint8Array {
  const encoder = new TextEncoder()
  const now = getDosDateTime(new Date())
  const localParts: Uint8Array[] = []
  const centralParts: Uint8Array[] = []
  let localLength = 0
  let centralLength = 0

  for (const file of files) {
    const name = encoder.encode(file.path)
    const content = encoder.encode(file.content)
    const checksum = crc32(content)
    const local = new Uint8Array(30 + name.length)
    const localView = new DataView(local.buffer)
    localView.setUint32(0, 0x04034b50, true)
    localView.setUint16(4, 20, true)
    localView.setUint16(6, 0x0800, true)
    localView.setUint16(10, now.time, true)
    localView.setUint16(12, now.date, true)
    localView.setUint32(14, checksum, true)
    localView.setUint32(18, content.length, true)
    localView.setUint32(22, content.length, true)
    localView.setUint16(26, name.length, true)
    local.set(name, 30)
    localParts.push(local, content)

    const central = new Uint8Array(46 + name.length)
    const centralView = new DataView(central.buffer)
    centralView.setUint32(0, 0x02014b50, true)
    centralView.setUint16(4, 20, true)
    centralView.setUint16(6, 20, true)
    centralView.setUint16(8, 0x0800, true)
    centralView.setUint16(12, now.time, true)
    centralView.setUint16(14, now.date, true)
    centralView.setUint32(16, checksum, true)
    centralView.setUint32(20, content.length, true)
    centralView.setUint32(24, content.length, true)
    centralView.setUint16(28, name.length, true)
    centralView.setUint32(42, localLength, true)
    central.set(name, 46)
    centralParts.push(central)
    centralLength += central.length
    localLength += local.length + content.length
  }

  const end = new Uint8Array(22)
  const endView = new DataView(end.buffer)
  endView.setUint32(0, 0x06054b50, true)
  endView.setUint16(8, files.length, true)
  endView.setUint16(10, files.length, true)
  endView.setUint32(12, centralLength, true)
  endView.setUint32(16, localLength, true)

  const result = new Uint8Array(localLength + centralLength + end.length)
  let offset = 0
  for (const part of [...localParts, ...centralParts, end]) {
    result.set(part, offset)
    offset += part.length
  }
  return result
}

export function buildUsageLogsXlsx(
  logs: UsageLog[],
  formatQuota: QuotaFormatter
): Uint8Array {
  const worksheet = buildWorksheet([
    Array.from(HEADERS),
    ...getRows(logs, formatQuota),
  ])
  return createZip([
    {
      path: '[Content_Types].xml',
      content: `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`,
    },
    {
      path: '_rels/.rels',
      content: `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
    },
    {
      path: 'xl/workbook.xml',
      content: `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="使用日志" sheetId="1" r:id="rId1"/></sheets></workbook>`,
    },
    {
      path: 'xl/_rels/workbook.xml.rels',
      content: `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`,
    },
    {
      path: 'xl/styles.xml',
      content: `<?xml version="1.0" encoding="UTF-8"?><styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><numFmts count="1"><numFmt numFmtId="164" formatCode="#0.00####"/></numFmts><fonts count="2"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><name val="Calibri"/></font></fonts><fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill></fills><borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders><cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs><cellXfs count="4"><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/><xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0" applyFont="true"/><xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0" applyAlignment="true"><alignment horizontal="left" vertical="top" wrapText="true"/></xf><xf numFmtId="164" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="true"/></cellXfs><cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles></styleSheet>`,
    },
    { path: 'xl/worksheets/sheet1.xml', content: worksheet },
  ])
}

export function downloadUsageLogsXlsx(content: Uint8Array): void {
  const buffer = new ArrayBuffer(content.byteLength)
  new Uint8Array(buffer).set(content)
  const blob = new Blob([buffer], {
    type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
  })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  const now = new Date()
  const pad = (value: number) => String(value).padStart(2, '0')
  link.href = url
  link.download = `usage_logs_${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}_${pad(now.getHours())}${pad(now.getMinutes())}${pad(now.getSeconds())}.xlsx`
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}
