/* Copyright (C) 2023-2026 QuantumNous */
import {
  Activity,
  ArrowUpRight,
  CircleGauge,
  Layers3,
  ShieldCheck,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { AnimateInView } from '@/components/animate-in-view'

const metrics = [
  { value: '50+', label: 'Provider connections', icon: Layers3, tone: 'mint' },
  { value: '99.98%', label: 'Route success', icon: ShieldCheck, tone: 'sky' },
  { value: '142ms', label: 'Latency p95', icon: CircleGauge, tone: 'violet' },
  { value: '24/7', label: 'Traffic visibility', icon: Activity, tone: 'mint' },
] as const

export function Stats() {
  const { t } = useTranslation()
  return (
    <AnimateInView
      as='section'
      className='relay-stats border-y px-6 py-7 md:py-8'
    >
      <div className='mx-auto grid max-w-7xl gap-3 sm:grid-cols-2 lg:grid-cols-4'>
        {metrics.map(({ value, label, icon: Icon, tone }) => (
          <div key={label} className={`relay-stat relay-stat--${tone}`}>
            <div className='relay-stat__icon'>
              <Icon className='size-4' />
            </div>
            <div className='min-w-0'>
              <div className='relay-stat__value'>{value}</div>
              <div className='relay-stat__label'>{t(label)}</div>
            </div>
            <ArrowUpRight className='relay-stat__arrow size-4' />
          </div>
        ))}
      </div>
    </AnimateInView>
  )
}
