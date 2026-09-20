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
  MATCH_RANGE,
  buildRequestRuleExpr,
  generateTimeTierExpr,
  parseTiersFromExpr,
  splitBillingExprAndRequestRules,
  tryParseRequestRuleExpr,
  tryParseTimeTierVisualConfig,
  type RequestRuleGroup,
  type TimeCondition,
} from './billing-expr.ts'
import { evalExprLocally, type ExtraTokenValues } from './tier-expr.ts'

const EMPTY_EXTRA_TOKENS: ExtraTokenValues = {
  cacheReadTokens: 0,
  cacheCreateTokens: 0,
  cacheCreate1hTokens: 0,
  imageTokens: 0,
  imageOutputTokens: 0,
  audioInputTokens: 0,
  audioOutputTokens: 0,
}

function range(
  timeFunc: TimeCondition['timeFunc'],
  start: string,
  end: string
): TimeCondition {
  return {
    source: 'time',
    timeFunc,
    timezone: 'Asia/Shanghai',
    mode: MATCH_RANGE,
    value: '',
    rangeStart: start,
    rangeEnd: end,
  }
}

describe('time-based billing rules', () => {
  test('uses AND for a same-day range and OR for an overnight range', () => {
    const sameDay = buildRequestRuleExpr([
      { conditions: [range('hour', '9', '12')], multiplier: '2' },
    ])
    const overnight = buildRequestRuleExpr([
      { conditions: [range('hour', '21', '6')], multiplier: '0.5' },
    ])

    assert.equal(
      sameDay,
      '(hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12 ? 2 : 1)'
    )
    assert.equal(
      overnight,
      '(hour("Asia/Shanghai") >= 21 || hour("Asia/Shanghai") < 6 ? 0.5 : 1)'
    )
  })

  test('builds and parses DeepSeek weekday peak periods without losing ranges', () => {
    const groups: RequestRuleGroup[] = [
      {
        conditions: [range('weekday', '1', '6'), range('hour', '9', '12')],
        multiplier: '2',
      },
      {
        conditions: [range('weekday', '1', '6'), range('hour', '14', '18')],
        multiplier: '2',
      },
    ]
    const expression = buildRequestRuleExpr(groups)
    const parsed = tryParseRequestRuleExpr(expression)

    assert.equal(parsed?.length, 2)
    assert.deepEqual(parsed, groups)
    assert.equal(buildRequestRuleExpr(parsed ?? []), expression)
  })

  test('rejects time values outside the function domain', () => {
    assert.equal(
      buildRequestRuleExpr([
        { conditions: [range('hour', '9', '24')], multiplier: '2' },
      ]),
      ''
    )
    assert.equal(
      buildRequestRuleExpr([
        {
          conditions: [range('weekday', '1', '6'), range('hour', '9', '24')],
          multiplier: '2',
        },
      ]),
      ''
    )
  })

  test('does not rewrite an impossible user-authored overnight AND condition', () => {
    const expression =
      '(hour("Asia/Shanghai") >= 21 && hour("Asia/Shanghai") < 6 ? 2 : 1)'
    const parsed = tryParseRequestRuleExpr(expression)

    assert.equal(parsed?.[0].conditions.length, 2)
    assert.equal(buildRequestRuleExpr(parsed ?? []), expression)
  })

  test('splits a multiline DeepSeek expression into pricing and time rules', () => {
    const expression = `
(tier("base", p * 0.15 + c * 0.6 + cr * 0.003))
* (weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") < 6
   && hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12 ? 2 : 1)
* (weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") < 6
   && hour("Asia/Shanghai") >= 14 && hour("Asia/Shanghai") < 18 ? 2 : 1)
`
    const split = splitBillingExprAndRequestRules(expression)

    assert.equal(
      split.billingExpr,
      'tier("base", p * 0.15 + c * 0.6 + cr * 0.003)'
    )
    assert.equal(tryParseRequestRuleExpr(split.requestRuleExpr)?.length, 2)
  })

  test('previews all DeepSeek peak boundaries in Asia/Shanghai', () => {
    const expression =
      'weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") < 6 && ((hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12) || (hour("Asia/Shanghai") >= 14 && hour("Asia/Shanghai") < 18))' +
      ' ? tier("peak", p * 0.3 + c * 1.2 + cr * 0.006)' +
      ' : tier("off_peak", p * 0.15 + c * 0.6 + cr * 0.003)'
    const cases = [
      ['Monday 08:59', '2026-09-21T00:59:00Z', 750_000],
      ['Monday 09:00', '2026-09-21T01:00:00Z', 1_500_000],
      ['Monday 11:59', '2026-09-21T03:59:00Z', 1_500_000],
      ['Monday 12:00', '2026-09-21T04:00:00Z', 750_000],
      ['Monday 13:59', '2026-09-21T05:59:00Z', 750_000],
      ['Monday 14:00', '2026-09-21T06:00:00Z', 1_500_000],
      ['Monday 17:59', '2026-09-21T09:59:00Z', 1_500_000],
      ['Monday 18:00', '2026-09-21T10:00:00Z', 750_000],
      ['Saturday 10:00', '2026-09-26T02:00:00Z', 750_000],
    ] as const

    for (const [name, timestamp, expectedCost] of cases) {
      const result = evalExprLocally(
        expression,
        1_000_000,
        1_000_000,
        EMPTY_EXTRA_TOKENS,
        new Date(timestamp)
      )
      assert.equal(result.error, null, name)
      assert.equal(result.cost, expectedCost, name)
      assert.equal(
        result.matchedTier,
        expectedCost === 1_500_000 ? 'peak' : 'off_peak',
        name
      )
    }
  })

  test('keeps existing request probes valid when preview context is absent', () => {
    const expression =
      '(tier("base", p * 2))' +
      ' * (param("service_tier") != nil ? 6 : 1)' +
      ' * (param("n") != nil && param("n") > 1 ? 2 : 1)' +
      ' * (param("tags") != nil && has(param("tags"), "fast") ? 3 : 1)' +
      ' * (header("x-priority") != "" ? 4 : 1)'
    const result = evalExprLocally(expression, 100, 0, EMPTY_EXTRA_TOKENS)

    assert.equal(result.error, null)
    assert.equal(result.cost, 200)
  })

  test('parses time-selected prices as peak and off-peak tiers', () => {
    const expression =
      'weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") < 6 && ((hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12) || (hour("Asia/Shanghai") >= 14 && hour("Asia/Shanghai") < 18))' +
      ' ? tier("peak", p * 0.3 + c * 1.2 + cr * 0.006)' +
      ' : tier("off_peak", p * 0.15 + c * 0.6 + cr * 0.003)'
    const tiers = parseTiersFromExpr(expression)

    assert.equal(tiers.length, 2)
    assert.deepEqual(
      tiers.map((tier) => ({
        label: tier.label,
        inputPrice: tier.inputPrice,
        outputPrice: tier.outputPrice,
        cacheReadPrice: tier.cacheReadPrice,
        isFallback: tier.isFallback ?? false,
      })),
      [
        {
          label: 'off_peak',
          inputPrice: 0.15,
          outputPrice: 0.6,
          cacheReadPrice: 0.003,
          isFallback: true,
        },
        {
          label: 'peak',
          inputPrice: 0.3,
          outputPrice: 1.2,
          cacheReadPrice: 0.006,
          isFallback: false,
        },
      ]
    )
    assert.match(tiers[1].conditionText ?? '', /09|hour/)

    const visualConfig = tryParseTimeTierVisualConfig(expression)
    assert.deepEqual(visualConfig, {
      timezone: 'Asia/Shanghai',
      weekdayStart: 1,
      weekdayEnd: 6,
      ranges: [
        { start: 9, end: 12 },
        { start: 14, end: 18 },
      ],
      matched: {
        label: 'peak',
        inputPrice: 0.3,
        outputPrice: 1.2,
        cacheReadPrice: 0.006,
      },
      fallback: {
        label: 'off_peak',
        inputPrice: 0.15,
        outputPrice: 0.6,
        cacheReadPrice: 0.003,
      },
    })
    assert.deepEqual(
      parseTiersFromExpr(generateTimeTierExpr(visualConfig!)).map(
        (tier) => tier.label
      ),
      ['off_peak', 'peak']
    )
  })

  test('does not present non-linear time tiers as simple token prices', () => {
    const condition = 'hour("UTC") < 12'
    for (const expression of [
      `${condition} ? tier("peak", p * 0.3 + max(c * 1.2, 1)) : tier("off_peak", p * 0.15 + c * 0.6)`,
      `${condition} ? tier("peak", p * 0.3 + 1) : tier("off_peak", p * 0.15 + c * 0.6)`,
      `${condition} ? tier("peak", p * 0.3 + p * 0.2) : tier("off_peak", p * 0.15 + c * 0.6)`,
      `${condition} ? tier("peak", p * 1e) : tier("off_peak", p * 0.15 + c * 0.6)`,
    ]) {
      assert.deepEqual(parseTiersFromExpr(expression), [])
    }
  })
})
