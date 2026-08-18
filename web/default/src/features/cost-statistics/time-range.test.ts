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
  MAX_NON_ADMIN_CUSTOM_RANGE_SECONDS,
  validateCostStatisticsCustomRange,
} from './time-range.ts'

describe('cost statistics custom time range', () => {
  test('allows a non-admin range of exactly 31 days', () => {
    assert.equal(
      validateCostStatisticsCustomRange(
        100,
        100 + MAX_NON_ADMIN_CUSTOM_RANGE_SECONDS,
        MAX_NON_ADMIN_CUSTOM_RANGE_SECONDS
      ),
      null
    )
  })

  test('rejects a non-admin range longer than 31 days', () => {
    assert.equal(
      validateCostStatisticsCustomRange(
        100,
        100 + MAX_NON_ADMIN_CUSTOM_RANGE_SECONDS + 1,
        MAX_NON_ADMIN_CUSTOM_RANGE_SECONDS
      ),
      'range_too_long'
    )
  })

  test('keeps admin custom ranges unlimited', () => {
    assert.equal(
      validateCostStatisticsCustomRange(
        100,
        100 + MAX_NON_ADMIN_CUSTOM_RANGE_SECONDS + 1
      ),
      null
    )
  })

  test('rejects invalid or reversed ranges for every role', () => {
    assert.equal(
      validateCostStatisticsCustomRange(
        0,
        100,
        MAX_NON_ADMIN_CUSTOM_RANGE_SECONDS
      ),
      'invalid_time'
    )
    assert.equal(
      validateCostStatisticsCustomRange(200, 100),
      'end_before_start'
    )
  })
})
