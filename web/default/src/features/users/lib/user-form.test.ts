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
  transformFormDataToPayload,
  userFormSchema,
  type UserFormValues,
} from './user-form.ts'

describe('user creation email', () => {
  test('validates and includes an optional email only when creating a user', () => {
    const values = {
      username: 'customer',
      display_name: 'Customer',
      password: 'password123',
      role: 1,
      email: ' customer@example.com ',
    } as UserFormValues

    const parsed = userFormSchema.parse(values)
    assert.equal(parsed.email, 'customer@example.com')
    assert.equal(
      transformFormDataToPayload(parsed).email,
      'customer@example.com'
    )
    assert.equal(transformFormDataToPayload(parsed, 10).email, undefined)
    assert.equal(
      userFormSchema.safeParse({ ...values, email: 'not-an-email' }).success,
      false
    )
    assert.equal(userFormSchema.parse({ ...values, email: '   ' }).email, '')
  })
})
