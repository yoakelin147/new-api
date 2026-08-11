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
import {
  MAX_STATISTICS_CUSTOM_RANGE_SECONDS,
  buildStatisticsTimeRange,
  validateStatisticsTimeRange,
} from './statistics-time-range.ts'

describe('dashboard statistics time range', () => {
  test('builds a rolling range ending at the supplied time', () => {
    const now = new Date('2026-08-11T12:00:00.000Z')

    assert.deepEqual(buildStatisticsTimeRange(7, now), {
      start_timestamp: 1_785_844_800,
      end_timestamp: 1_786_449_600,
    })
  })

  test('rejects a range whose end precedes its start', () => {
    assert.equal(
      validateStatisticsTimeRange({
        start_timestamp: 200,
        end_timestamp: 100,
      }),
      'end_before_start'
    )
  })

  test('rejects a range longer than 183 days', () => {
    assert.equal(
      validateStatisticsTimeRange({
        start_timestamp: 100,
        end_timestamp: 100 + MAX_STATISTICS_CUSTOM_RANGE_SECONDS + 1,
      }),
      'range_too_long'
    )
  })

  test('accepts the maximum 183-day range', () => {
    assert.equal(
      validateStatisticsTimeRange({
        start_timestamp: 100,
        end_timestamp: 100 + MAX_STATISTICS_CUSTOM_RANGE_SECONDS,
      }),
      null
    )
  })
})
