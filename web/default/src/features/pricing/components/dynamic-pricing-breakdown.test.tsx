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
import { createElement } from 'react'
import { createInstance } from 'i18next'
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider, initReactI18next } from 'react-i18next'
import { TieredPricingEditor } from '@/features/system-settings/models/tiered-pricing-editor.tsx'
import { DynamicPricingBreakdown } from './dynamic-pricing-breakdown.tsx'

async function createTestI18n() {
  const i18n = createInstance()
  await i18n.use(initReactI18next).init({
    lng: 'en',
    resources: { en: { translation: {} } },
  })
  return i18n
}

const DEEPSEEK_EXPRESSION =
  'weekday("Asia/Shanghai") >= 1 && weekday("Asia/Shanghai") < 6 && ((hour("Asia/Shanghai") >= 9 && hour("Asia/Shanghai") < 12) || (hour("Asia/Shanghai") >= 14 && hour("Asia/Shanghai") < 18))' +
  ' ? tier("peak", p * 0.3 + c * 1.2 + cr * 0.006)' +
  ' : tier("off_peak", p * 0.15 + c * 0.6 + cr * 0.003)'

test('renders DeepSeek peak and off-peak tiers with their schedule', async () => {
  const i18n = await createTestI18n()
  const html = renderToStaticMarkup(
    createElement(
      I18nextProvider,
      { i18n },
      createElement(DynamicPricingBreakdown, {
        billingExpr: DEEPSEEK_EXPRESSION,
      })
    )
  )

  for (const text of [
    'off_peak',
    'peak',
    'Other times',
    'Monday to Friday',
    '09:00',
    '12:00',
    '14:00',
    '18:00',
  ]) {
    assert.match(html, new RegExp(text))
  }
})

test('opens a time-tier expression in visual mode without replacing it with zero pricing', async () => {
  const i18n = await createTestI18n()
  const html = renderToStaticMarkup(
    createElement(
      I18nextProvider,
      { i18n },
      createElement(TieredPricingEditor, {
        modelName: 'deepseek-flash',
        billingExpr: DEEPSEEK_EXPRESSION,
        requestRuleExpr: '',
        onBillingExprChange: () => {},
        onRequestRuleExprChange: () => {},
      })
    )
  )

  assert.match(html, /value="off_peak"/)
  assert.match(html, /value="peak"/)
  assert.match(html, /Asia\/Shanghai/)
  assert.doesNotMatch(html, /p \* 0 \+ c \* 0/)
})
