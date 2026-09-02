/* Copyright (C) 2023-2026 QuantumNous */
import {
  Activity,
  ArrowUpRight,
  CircleGauge,
  Layers3,
  ShieldCheck,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useCountUp } from '@/hooks/use-count-up'
import { useSpotlight } from '@/hooks/use-spotlight'
import { AnimateInView } from '@/components/animate-in-view'

const metrics = [
  {
    value: '50+',
    label: 'Provider connections',
    icon: Layers3,
    tone: 'mint',
    numericValue: 50,
    suffix: '+',
    decimals: 0,
  },
  {
    value: '99.98%',
    label: 'Route success',
    icon: ShieldCheck,
    tone: 'sky',
    numericValue: 99.98,
    suffix: '%',
    decimals: 2,
  },
  {
    value: '142ms',
    label: 'Latency p95',
    icon: CircleGauge,
    tone: 'violet',
    numericValue: 142,
    suffix: 'ms',
    decimals: 0,
  },
  {
    value: '24/7',
    label: 'Traffic visibility',
    icon: Activity,
    tone: 'mint',
  },
] as const

function StatCard(props: { metric: (typeof metrics)[number]; index: number }) {
  const { t } = useTranslation()
  const spotlightRef = useSpotlight<HTMLDivElement>()
  const numericValue =
    'numericValue' in props.metric ? props.metric.numericValue : undefined
  const suffix = 'suffix' in props.metric ? props.metric.suffix : ''
  const decimals = 'decimals' in props.metric ? props.metric.decimals : 0
  const { display } = useCountUp<HTMLDivElement>(numericValue ?? Number.NaN, {
    decimals,
    delay: props.index * 120,
  })
  const Icon = props.metric.icon
  return (
    <AnimateInView
      className={`relay-stat relay-stat--${props.metric.tone} relay-spotlight`}
      delay={props.index * 90}
      animation='fade-up'
    >
      <div ref={spotlightRef} className='contents'>
        <div className='relay-stat__icon'>
          <Icon className='size-4' />
        </div>
        <div className='min-w-0'>
          <div className='relay-stat__value'>
            {numericValue !== undefined
              ? `${display}${suffix}`
              : props.metric.value}
          </div>
          <div className='relay-stat__label'>{t(props.metric.label)}</div>
        </div>
        <ArrowUpRight className='relay-stat__arrow size-4' />
      </div>
    </AnimateInView>
  )
}

export function Stats() {
  return (
    <section className='relay-stats border-y px-6 py-7 md:py-8'>
      <div className='mx-auto grid max-w-7xl gap-3 sm:grid-cols-2 lg:grid-cols-4'>
        {metrics.map((metric, index) => (
          <StatCard key={metric.label} metric={metric} index={index} />
        ))}
      </div>
    </section>
  )
}
