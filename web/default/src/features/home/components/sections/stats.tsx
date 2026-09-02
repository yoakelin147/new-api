/* Copyright (C) 2023-2026 QuantumNous */
import { Boxes, Globe2, ShieldCheck, Zap } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useCountUp } from '@/hooks/use-count-up'
import { useSpotlight } from '@/hooks/use-spotlight'
import { AnimateInView } from '@/components/animate-in-view'

const metrics = [
  {
    label: 'Model providers',
    icon: Globe2,
    tone: 'mint',
    numericValue: 50,
    suffix: '+',
    decimals: 0,
  },
  {
    label: 'Models available',
    icon: Boxes,
    tone: 'sky',
    numericValue: 300,
    suffix: '+',
    decimals: 0,
  },
  {
    label: 'Latency p95',
    icon: Zap,
    tone: 'violet',
    numericValue: 90,
    suffix: 'ms',
    decimals: 0,
  },
  {
    label: 'Service uptime',
    icon: ShieldCheck,
    tone: 'mint',
    numericValue: 99.98,
    suffix: '%',
    decimals: 2,
  },
] as const

function StatCard(props: { metric: (typeof metrics)[number]; index: number }) {
  const { t } = useTranslation()
  const spotlightRef = useSpotlight<HTMLDivElement>()
  const { ref, display } = useCountUp<HTMLDivElement>(
    props.metric.numericValue,
    {
      decimals: props.metric.decimals,
      delay: props.index * 120,
    }
  )
  const Icon = props.metric.icon
  return (
    <AnimateInView
      className={`nexus-stat nexus-stat--${props.metric.tone} relay-spotlight`}
      delay={props.index * 90}
      animation='fade-up'
    >
      <div ref={spotlightRef} className='contents'>
        <div ref={ref} className='contents'>
          <div className='nexus-stat__icon'>
            <Icon className='size-4' />
          </div>
          <div className='min-w-0'>
            <div className='nexus-stat__value'>
              {display}
              {props.metric.suffix}
            </div>
            <div className='nexus-stat__label'>{t(props.metric.label)}</div>
          </div>
        </div>
      </div>
    </AnimateInView>
  )
}

export function Stats() {
  return (
    <section className='nexus-stats border-y px-6 py-8 md:py-10'>
      <div className='mx-auto grid max-w-7xl gap-3 sm:grid-cols-2 lg:grid-cols-4'>
        {metrics.map((metric, index) => (
          <StatCard key={metric.label} metric={metric} index={index} />
        ))}
      </div>
    </section>
  )
}
