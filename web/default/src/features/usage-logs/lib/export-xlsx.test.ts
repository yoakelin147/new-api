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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { buildUsageLogsXlsx } from './export-xlsx.ts'

describe('usage log XLSX export', () => {
  test('exports customer-facing fields without raw admin metadata', () => {
    const content = buildUsageLogsXlsx(
      [
        {
          id: 1,
          user_id: 7,
          created_at: 1_700_000_000,
          type: 2,
          content: '<completed>&\u0001',
          username: 'customer',
          token_name: 'customer-token',
          model_name: 'gpt-export',
          quota: 250,
          prompt_tokens: 120,
          completion_tokens: 30,
          use_time: 4,
          is_stream: true,
          channel: 3,
          channel_name: 'internal-channel',
          token_id: 9,
          group: 'default',
          ip: '192.0.2.1',
          request_id: 'request-visible',
          upstream_request_id: 'upstream-visible',
          other: JSON.stringify({
            cache_tokens: 20,
            frt: 1.2,
            billing_mode: 'tiered_expr',
            matched_tier: 'standard',
            admin_info: { secret: 'must-not-export' },
          }),
        },
      ],
      () => '¥0.0035'
    )

    assert.deepEqual(Array.from(content.slice(0, 2)), [0x50, 0x4b])
    const rawArchive = new TextDecoder().decode(content)
    assert.match(rawArchive, /&lt;completed&gt;&amp;/)
    assert.match(rawArchive, /request-visible/)
    assert.match(rawArchive, /upstream-visible/)
    assert.match(rawArchive, /¥0\.0035/)
    assert.doesNotMatch(rawArchive, /must-not-export|admin_info|192\.0\.2\.1/)
  })
})
