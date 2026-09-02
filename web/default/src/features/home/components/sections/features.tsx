/* Copyright (C) 2023-2026 QuantumNous */
import {
  ArrowUpRight,
  BarChart3,
  KeyRound,
  Network,
  ShieldCheck,
  SlidersHorizontal,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useSpotlight } from '@/hooks/use-spotlight'
import { AnimateInView } from '@/components/animate-in-view'

const featureCards = [
  {
    number: '01',
    title: 'Easy to connect',
    copy: 'Use the tools and models you already know, with one simple connection.',
    icon: Network,
    tone: 'mint',
  },
  {
    number: '02',
    title: 'Always available',
    copy: 'Enjoy a dependable service that keeps your work moving when demand changes.',
    icon: SlidersHorizontal,
    tone: 'sky',
  },
  {
    number: '03',
    title: 'Clear and fair pricing',
    copy: 'See what you use and what you pay, with straightforward usage details.',
    icon: BarChart3,
    tone: 'violet',
  },
  {
    number: '04',
    title: 'Made for real work',
    copy: 'From coding to content and images, get the right capability for every task.',
    icon: KeyRound,
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
        className={`relay-feature-card relay-feature-card--${card.tone} relay-spotlight h-full`}
      >
        <div className='flex items-start justify-between gap-6'>
          <div className='relay-feature-card__number'>{card.number}</div>
          <Icon className='relay-feature-card__icon size-5' />
        </div>
        <h3 className='relay-feature-card__title mt-12'>{t(card.title)}</h3>
        <p className='relay-feature-card__copy mt-3 max-w-sm text-sm leading-7'>
          {t(card.copy)}
        </p>
        <ArrowUpRight className='relay-feature-card__arrow mt-8 size-5' />
      </article>
    </AnimateInView>
  )
}

export function Features() {
  const { t } = useTranslation()
  return (
    <AnimateInView as='section' className='relay-features px-6 py-24 md:py-32'>
      <div className='mx-auto max-w-7xl'>
        <div className='relay-section-heading mb-12 flex flex-col justify-between gap-6 md:flex-row md:items-end'>
          <div>
            <p className='relay-section-kicker'>
              {t('Everything you need to create')}
            </p>
            <h2 className='relay-section-title mt-3'>{t('with AI, simply')}</h2>
          </div>
          <p className='relay-section-intro max-w-md text-sm leading-7'>
            {t(
              'A simple, dependable way to access powerful AI services without managing the complexity behind them.'
            )}
          </p>
        </div>
        <div className='grid gap-3 md:grid-cols-2'>
          {featureCards.map((card, index) => (
            <FeatureCard key={card.number} card={card} index={index} />
          ))}
        </div>
        <div className='relay-capability-strip mt-3 grid gap-3 sm:grid-cols-3'>
          <div>
            <ShieldCheck className='size-4' />
            <span>{t('Stable service')}</span>
          </div>
          <div>
            <Network className='size-4' />
            <span>{t('Flexible access')}</span>
          </div>
          <div>
            <BarChart3 className='size-4' />
            <span>{t('Clear pricing')}</span>
          </div>
        </div>
      </div>
    </AnimateInView>
  )
}
