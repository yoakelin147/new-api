/* Copyright (C) 2023-2026 QuantumNous */
import {
  ArrowUpRight,
  BarChart3,
  GitBranch,
  KeyRound,
  ShieldCheck,
  Workflow,
  Zap,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useSpotlight } from '@/hooks/use-spotlight'
import { AnimateInView } from '@/components/animate-in-view'

const featureCards = [
  {
    number: '01',
    title: 'Unified API interface',
    copy: 'One OpenAI-compatible endpoint for chat, embeddings, images, audio and video. Switch models by changing a single parameter.',
    icon: Workflow,
    tone: 'mint',
  },
  {
    number: '02',
    title: 'Smart load balancing',
    copy: 'Distribute traffic across providers by latency, cost, and availability. Automatic failover keeps your app running when a provider degrades.',
    icon: GitBranch,
    tone: 'sky',
  },
  {
    number: '03',
    title: 'Transparent cost control',
    copy: 'Real-time token metering, per-key budgets, and detailed usage analytics. Know exactly what every request costs before you send it.',
    icon: BarChart3,
    tone: 'violet',
  },
  {
    number: '04',
    title: 'Enterprise-grade security',
    copy: 'Isolated API keys, granular permissions, rate limiting and audit logs. Your credentials and data stay under your control.',
    icon: ShieldCheck,
    tone: 'mint',
  },
] as const

function FeatureCard({
  card,
  index,
}: {
  card: (typeof featureCards)[number]
  index: number
}) {
  const { t } = useTranslation()
  const spotlightRef = useSpotlight<HTMLElement>()
  const Icon = card.icon
  return (
    <AnimateInView
      as='div'
      delay={index * 100}
      animation='fade-up'
      className='h-full'
    >
      <article
        ref={spotlightRef}
        className={`nexus-feature-card nexus-feature-card--${card.tone} relay-spotlight h-full`}
      >
        <div className='flex items-start justify-between gap-6'>
          <div className='nexus-feature-card__number'>{card.number}</div>
          <Icon className='nexus-feature-card__icon size-5' />
        </div>
        <h3 className='nexus-feature-card__title mt-12'>{t(card.title)}</h3>
        <p className='nexus-feature-card__copy mt-3 max-w-sm text-sm leading-7'>
          {t(card.copy)}
        </p>
        <ArrowUpRight className='nexus-feature-card__arrow mt-8 size-5' />
      </article>
    </AnimateInView>
  )
}

export function Features() {
  const { t } = useTranslation()
  return (
    <AnimateInView as='section' className='nexus-features px-6 py-24 md:py-32'>
      <div className='mx-auto max-w-7xl'>
        <div className='nexus-section-heading mb-12 flex flex-col justify-between gap-6 md:flex-row md:items-end'>
          <div>
            <p className='nexus-section-kicker'>
              {t('Built for production AI')}
            </p>
            <h2 className='nexus-section-title mt-3'>
              {t('Everything you need to ship with confidence')}
            </h2>
          </div>
          <p className='nexus-section-intro max-w-md text-sm leading-7'>
            {t(
              'Stop juggling API keys and provider quirks. One integration gives you the entire model ecosystem with the reliability your users expect.'
            )}
          </p>
        </div>
        <div className='grid gap-3 md:grid-cols-2'>
          {featureCards.map((card, index) => (
            <FeatureCard key={card.number} card={card} index={index} />
          ))}
        </div>
        <div className='nexus-capability-strip mt-3 grid gap-3 sm:grid-cols-3'>
          <div>
            <Zap className='size-4' />
            <span>{t('Millisecond routing')}</span>
          </div>
          <div>
            <KeyRound className='size-4' />
            <span>{t('Fine-grained access')}</span>
          </div>
          <div>
            <BarChart3 className='size-4' />
            <span>{t('Usage analytics')}</span>
          </div>
        </div>
      </div>
    </AnimateInView>
  )
}
