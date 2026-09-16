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
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { testClickHouseAlertWebhook } from '../api'
import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'
import { safeNumberFieldProps } from '../utils/numeric-field'
import {
  createFaultHandlingSchema,
  PROBE_INTERVAL_MAX_SECONDS,
  PROBE_INTERVAL_MIN_SECONDS,
  PROBE_TIMEOUT_MAX_MILLISECONDS,
  PROBE_TIMEOUT_MIN_MILLISECONDS,
} from './fault-handling'

type FaultHandlingDefaults = {
  'fault_handling.clickhouse_auto_recovery_enabled': boolean
  'fault_handling.clickhouse_probe_interval_seconds': number
  'fault_handling.clickhouse_probe_timeout_milliseconds': number
  'fault_handling.clickhouse_alert_enabled': boolean
  'fault_handling.clickhouse_alert_webhook_configured': boolean
}

export function FaultHandlingSection(props: {
  defaultValues: FaultHandlingDefaults
}) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [testingWebhook, setTestingWebhook] = useState(false)
  const schema = createFaultHandlingSchema(t)
  type FormValues = ReturnType<typeof schema.parse>

  const formDefaults: FormValues = {
    fault_handling: {
      clickhouse_auto_recovery_enabled:
        props.defaultValues['fault_handling.clickhouse_auto_recovery_enabled'],
      clickhouse_probe_interval_seconds:
        props.defaultValues['fault_handling.clickhouse_probe_interval_seconds'],
      clickhouse_probe_timeout_milliseconds:
        props.defaultValues[
          'fault_handling.clickhouse_probe_timeout_milliseconds'
        ],
      clickhouse_alert_enabled:
        props.defaultValues['fault_handling.clickhouse_alert_enabled'],
      clickhouse_alert_webhook_configured:
        props.defaultValues[
          'fault_handling.clickhouse_alert_webhook_configured'
        ],
      clickhouse_alert_webhook_url: '',
    },
  }
  const form = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: formDefaults,
  })
  useResetForm(form, formDefaults)

  const autoRecoveryEnabled = form.watch(
    'fault_handling.clickhouse_auto_recovery_enabled'
  )
  const webhookConfigured = form.watch(
    'fault_handling.clickhouse_alert_webhook_configured'
  )
  const onSubmit = async (values: FormValues) => {
    const normalized: FaultHandlingDefaults = {
      'fault_handling.clickhouse_auto_recovery_enabled':
        values.fault_handling.clickhouse_auto_recovery_enabled,
      'fault_handling.clickhouse_probe_interval_seconds':
        values.fault_handling.clickhouse_probe_interval_seconds,
      'fault_handling.clickhouse_probe_timeout_milliseconds':
        values.fault_handling.clickhouse_probe_timeout_milliseconds,
      'fault_handling.clickhouse_alert_enabled':
        values.fault_handling.clickhouse_alert_enabled,
      'fault_handling.clickhouse_alert_webhook_configured':
        values.fault_handling.clickhouse_alert_webhook_configured,
    }
    const updates = (
      Object.keys(normalized) as Array<keyof FaultHandlingDefaults>
    ).filter((key) => normalized[key] !== props.defaultValues[key])

    const webhookURL =
      values.fault_handling.clickhouse_alert_webhook_url?.trim()
    if (updates.length === 0 && !webhookURL) {
      toast.info(t('No changes to save'))
      return
    }
    for (const key of updates) {
      await updateOption.mutateAsync({ key, value: normalized[key] })
    }
    if (webhookURL) {
      await updateOption.mutateAsync({
        key: 'fault_handling.clickhouse_alert_webhook_url',
        value: webhookURL,
      })
    }
  }

  const clearWebhook = async () => {
    await updateOption.mutateAsync({
      key: 'fault_handling.clickhouse_alert_webhook_url',
      value: '',
    })
    form.reset(formDefaults)
  }

  const testWebhook = async () => {
    setTestingWebhook(true)
    try {
      const result = await testClickHouseAlertWebhook()
      if (result.success) toast.success(t('测试消息已发送'))
      else toast.error(result.message || t('测试消息发送失败'))
    } catch (error) {
      toast.error(
        error instanceof Error ? error.message : t('测试消息发送失败')
      )
    } finally {
      setTestingWebhook(false)
    }
  }

  return (
    <SettingsSection title={t('故障处理')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />
          <FormField
            control={form.control}
            name='fault_handling.clickhouse_auto_recovery_enabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('启用 ClickHouse 自动恢复探测')}</FormLabel>
                  <FormDescription>
                    {t(
                      '后台任务按配置间隔探测 ClickHouse 可用性，业务查询不承担恢复探测'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />
          <FormField
            control={form.control}
            name='fault_handling.clickhouse_alert_enabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('启用企业微信故障告警')}</FormLabel>
                  <FormDescription>
                    {t('只在日志读写切换或恢复时发送一次告警')}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />
          <FormField
            control={form.control}
            name='fault_handling.clickhouse_alert_webhook_url'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('企业微信机器人 Webhook')}</FormLabel>
                <FormControl>
                  <Input
                    type='password'
                    autoComplete='new-password'
                    placeholder={
                      webhookConfigured
                        ? t('已配置，留空保持不变')
                        : t('粘贴企业微信 Webhook 地址')
                    }
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {webhookConfigured
                    ? t('Webhook 已配置，地址不会回显')
                    : t('仅支持企业微信 https Webhook 地址')}
                </FormDescription>
                <FormMessage />
                {webhookConfigured && (
                  <div className='flex flex-wrap gap-2'>
                    <Button
                      type='button'
                      variant='outline'
                      size='sm'
                      onClick={testWebhook}
                      disabled={updateOption.isPending || testingWebhook}
                    >
                      {t('发送测试消息')}
                    </Button>
                    <Button
                      type='button'
                      variant='outline'
                      size='sm'
                      onClick={clearWebhook}
                      disabled={updateOption.isPending || testingWebhook}
                    >
                      {t('清除 Webhook')}
                    </Button>
                  </div>
                )}
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name='fault_handling.clickhouse_probe_interval_seconds'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('自动探测间隔（秒）')}</FormLabel>
                <FormControl>
                  <Input
                    type='number'
                    min={PROBE_INTERVAL_MIN_SECONDS}
                    max={PROBE_INTERVAL_MAX_SECONDS}
                    step={1}
                    disabled={!autoRecoveryEnabled}
                    {...safeNumberFieldProps(field)}
                  />
                </FormControl>
                <FormDescription>
                  {t('允许范围：5–3600 秒，默认 30 秒')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name='fault_handling.clickhouse_probe_timeout_milliseconds'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('单次探测超时（毫秒）')}</FormLabel>
                <FormControl>
                  <Input
                    type='number'
                    min={PROBE_TIMEOUT_MIN_MILLISECONDS}
                    max={PROBE_TIMEOUT_MAX_MILLISECONDS}
                    step={100}
                    disabled={!autoRecoveryEnabled}
                    {...safeNumberFieldProps(field)}
                  />
                </FormControl>
                <FormDescription>
                  {t('允许范围：100–10000 毫秒，默认 1000 毫秒')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
