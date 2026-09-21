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
import { RotateCcw, SlidersHorizontal } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import { Switch } from '@/components/ui/switch'
import {
  sideDrawerContentClassName,
  sideDrawerFooterClassName,
  sideDrawerFormClassName,
  sideDrawerHeaderClassName,
} from '@/components/drawer-layout'
import type { ParameterEnabled, PlaygroundConfig } from '../types'

type ParameterKey = keyof ParameterEnabled

type ParameterDefinition = {
  key: ParameterKey
  label: string
  min: number
  max: number
  step: number
}

const PARAMETERS: ParameterDefinition[] = [
  {
    key: 'temperature',
    label: 'Temperature',
    min: 0,
    max: 2,
    step: 0.1,
  },
  {
    key: 'top_p',
    label: 'Top P',
    min: 0,
    max: 1,
    step: 0.05,
  },
  {
    key: 'max_tokens',
    label: 'Max Tokens',
    min: 1,
    max: 1048576,
    step: 1,
  },
  {
    key: 'frequency_penalty',
    label: 'Frequency Penalty',
    min: -2,
    max: 2,
    step: 0.1,
  },
  {
    key: 'presence_penalty',
    label: 'Presence Penalty',
    min: -2,
    max: 2,
    step: 0.1,
  },
  {
    key: 'seed',
    label: 'Seed',
    min: 0,
    max: 2147483647,
    step: 1,
  },
]

interface PlaygroundParametersProps {
  config: PlaygroundConfig
  parameterEnabled: ParameterEnabled
  disabled?: boolean
  onConfigChange: <K extends keyof PlaygroundConfig>(
    key: K,
    value: PlaygroundConfig[K]
  ) => void
  onParameterEnabledChange: (key: ParameterKey, value: boolean) => void
  onReset: () => void
}

export function PlaygroundParameters({
  config,
  parameterEnabled,
  disabled,
  onConfigChange,
  onParameterEnabledChange,
  onReset,
}: PlaygroundParametersProps) {
  const { t } = useTranslation()

  return (
    <Sheet>
      <SheetTrigger
        render={
          <Button
            type='button'
            variant='outline'
            size='sm'
            className='h-8 gap-2 px-2.5'
            disabled={disabled}
            aria-label={t('Parameters')}
          />
        }
      >
        <SlidersHorizontal className='h-4 w-4' aria-hidden='true' />
        <span className='hidden sm:inline'>{t('Parameters')}</span>
      </SheetTrigger>
      <SheetContent className={sideDrawerContentClassName('sm:max-w-md')}>
        <SheetHeader className={sideDrawerHeaderClassName()}>
          <SheetTitle>{t('Parameters')}</SheetTitle>
          <SheetDescription>
            {t('Configure request parameters')}
          </SheetDescription>
        </SheetHeader>

        <div className={sideDrawerFormClassName()}>
          <div className='flex items-center justify-between gap-4 border-b pb-4'>
            <div>
              <div className='text-sm font-medium'>{t('Stream')}</div>
            </div>
            <Switch
              checked={config.stream}
              onCheckedChange={(checked) => onConfigChange('stream', checked)}
              aria-label={t('Stream')}
            />
          </div>

          {PARAMETERS.map((parameter) => {
            const enabled = parameterEnabled[parameter.key]
            const value = config[parameter.key] ?? 0
            return (
              <div key={parameter.key} className='space-y-3 border-b pb-4'>
                <div className='flex items-center justify-between gap-4'>
                  <div>
                    <label
                      htmlFor={`playground-${parameter.key}`}
                      className='text-sm font-medium'
                    >
                      {t(parameter.label)}
                    </label>
                  </div>
                  <Switch
                    checked={enabled}
                    onCheckedChange={(checked) => {
                      if (checked && config[parameter.key] === null) {
                        onConfigChange(parameter.key, 0 as never)
                      }
                      onParameterEnabledChange(parameter.key, checked)
                    }}
                    aria-label={t(parameter.label)}
                  />
                </div>
                <Input
                  id={`playground-${parameter.key}`}
                  type='number'
                  value={value}
                  min={parameter.min}
                  max={parameter.max}
                  step={parameter.step}
                  disabled={!enabled}
                  onChange={(event) => {
                    const nextValue = Number(event.target.value)
                    if (!Number.isFinite(nextValue)) return
                    const boundedValue = Math.min(
                      parameter.max,
                      Math.max(parameter.min, nextValue)
                    )
                    onConfigChange(
                      parameter.key,
                      (parameter.key === 'seed'
                        ? Math.trunc(boundedValue)
                        : boundedValue) as never
                    )
                  }}
                />
              </div>
            )
          })}
        </div>

        <SheetFooter className={sideDrawerFooterClassName('grid-cols-1')}>
          <Button type='button' variant='outline' onClick={onReset}>
            <RotateCcw className='mr-2 h-4 w-4' aria-hidden='true' />
            {t('Reset')}
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
