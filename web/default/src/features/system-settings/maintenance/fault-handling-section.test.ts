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
import { createFaultHandlingSchema } from './fault-handling.ts'

const schema = createFaultHandlingSchema((key) => key)

function values(interval: number, timeout: number) {
  return {
    fault_handling: {
      clickhouse_auto_recovery_enabled: true,
      clickhouse_probe_interval_seconds: interval,
      clickhouse_probe_timeout_milliseconds: timeout,
      clickhouse_alert_enabled: false,
      clickhouse_alert_webhook_url: '',
      clickhouse_alert_webhook_configured: false,
    },
  }
}

describe('fault handling settings limits', () => {
  test('accepts boundary values', () => {
    assert.equal(schema.safeParse(values(5, 100)).success, true)
    assert.equal(schema.safeParse(values(3600, 10000)).success, true)
  })

  test('rejects values outside the safe range', () => {
    assert.equal(schema.safeParse(values(4, 100)).success, false)
    assert.equal(schema.safeParse(values(3601, 100)).success, false)
    assert.equal(schema.safeParse(values(5, 99)).success, false)
    assert.equal(schema.safeParse(values(5, 10001)).success, false)
  })
})
