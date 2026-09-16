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
import * as z from 'zod'

export const PROBE_INTERVAL_MIN_SECONDS = 5
export const PROBE_INTERVAL_MAX_SECONDS = 3600
export const PROBE_TIMEOUT_MIN_MILLISECONDS = 100
export const PROBE_TIMEOUT_MAX_MILLISECONDS = 10000

export const createFaultHandlingSchema = (t: (key: string) => string) =>
  z.object({
    fault_handling: z.object({
      clickhouse_auto_recovery_enabled: z.boolean(),
      clickhouse_probe_interval_seconds: z
        .number()
        .int()
        .min(
          PROBE_INTERVAL_MIN_SECONDS,
          t('The probe interval must be between 5 and 3600 seconds')
        )
        .max(
          PROBE_INTERVAL_MAX_SECONDS,
          t('The probe interval must be between 5 and 3600 seconds')
        ),
      clickhouse_probe_timeout_milliseconds: z
        .number()
        .int()
        .min(
          PROBE_TIMEOUT_MIN_MILLISECONDS,
          t('The probe timeout must be between 100 and 10000 milliseconds')
        )
        .max(
          PROBE_TIMEOUT_MAX_MILLISECONDS,
          t('The probe timeout must be between 100 and 10000 milliseconds')
        ),
      clickhouse_alert_enabled: z.boolean(),
      clickhouse_alert_webhook_url: z.string().optional(),
      clickhouse_alert_webhook_configured: z.boolean(),
    }),
  })
