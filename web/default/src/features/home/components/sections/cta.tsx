/* Copyright (C) 2023-2026 QuantumNous */
import { Link } from '@tanstack/react-router'
import { ArrowRight, RadioTower } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { AnimateInView } from '@/components/animate-in-view'

interface CTAProps {
  isAuthenticated?: boolean
}

export function CTA({ isAuthenticated }: CTAProps) {
  const { t } = useTranslation()
  if (isAuthenticated) return null
  return (
    <AnimateInView as='section' className='relay-cta px-6 py-20 md:py-28'>
      <div className='relay-cta__inner mx-auto flex max-w-7xl flex-col justify-between gap-10 md:flex-row md:items-end'>
        <div className='max-w-2xl'>
          <div className='relay-cta__icon'>
            <RadioTower className='size-5' />
          </div>
          <p className='relay-section-kicker mt-6'>
            {t('Ready to get started?')}
          </p>
          <h2 className='relay-cta__title mt-3'>
            {t('Bring your next idea to life.')}
          </h2>
          <p className='relay-cta__copy mt-5 max-w-xl text-sm leading-7'>
            {t(
              'Access dependable AI services with a simple setup, clear pricing, and the freedom to focus on your work.'
            )}
          </p>
        </div>
        <div className='flex flex-wrap gap-3'>
          <Button
            className='relay-cta__primary'
            render={
              <Link to='/dashboard/$section' params={{ section: 'overview' }} />
            }
          >
            {t('Get Started')} <ArrowRight className='size-4' />
          </Button>
          <Button
            variant='outline'
            className='relay-cta__secondary'
            render={<Link to='/pricing' />}
          >
            {t('View Pricing')}
          </Button>
        </div>
      </div>
    </AnimateInView>
  )
}
