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
import { getHealthStatus, successRateStatus } from './health.ts'

describe('health status success-rate thresholds', () => {
  test('marks success rates of 90% or higher as healthy', () => {
    assert.equal(successRateStatus(90), 'healthy')
    assert.equal(successRateStatus(100), 'healthy')
    const highLatencyRow = { success_rate: 100, avg_latency_ms: 76_000 }
    assert.equal(getHealthStatus(highLatencyRow), 'healthy')
  })

  test('marks success rates from 60% up to 90% as warning', () => {
    assert.equal(successRateStatus(60), 'warning')
    assert.equal(successRateStatus(89.999), 'warning')
  })

  test('marks success rates below 60% as critical', () => {
    assert.equal(successRateStatus(59.999), 'critical')
    assert.equal(successRateStatus(0), 'critical')
  })
})
