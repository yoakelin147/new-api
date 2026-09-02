/* Copyright (C) 2023-2026 QuantumNous */
import { Link } from '@tanstack/react-router'
import { ArrowRight, Sparkles } from 'lucide-react'
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
    <AnimateInView as='section' className='nexus-cta px-6 py-20 md:py-28'>
      <div className='nexus-cta__inner mx-auto flex max-w-7xl flex-col justify-between gap-10 md:flex-row md:items-end'>
        <div className='max-w-2xl'>
          <div className='nexus-cta__icon'>
            <Sparkles className='size-5' />
          </div>
          <p className='nexus-section-kicker mt-6'>
            {t('Ready to simplify your AI stack?')}
          </p>
          <h2 className='nexus-cta__title mt-3'>
            {t('Organized access to every model you need.')}
          </h2>
          <p className='nexus-cta__copy mt-5 max-w-xl text-sm leading-7'>
            {t(
              'Join thousands of developers building on a single, reliable API gateway. Start free, scale as you grow.'
            )}
          </p>
        </div>
        <div className='flex flex-wrap gap-3'>
          <Button
            className='nexus-cta__primary'
            render={
              <Link to='/dashboard/$section' params={{ section: 'overview' }} />
            }
          >
            {t('Start Building')} <ArrowRight className='size-4' />
          </Button>
          <Button
            variant='outline'
            className='nexus-cta__secondary'
            render={<Link to='/pricing' />}
          >
            {t('View Pricing')}
          </Button>
        </div>
      </div>
    </AnimateInView>
  )
}
