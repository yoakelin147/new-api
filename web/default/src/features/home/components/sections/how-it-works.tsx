/* Copyright (C) 2023-2026 QuantumNous */
import { ArrowRight, Code2, KeyRound, Rocket } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { AnimateInView } from '@/components/animate-in-view'

const steps = [
  {
    number: '01',
    title: 'Get your API key',
    copy: 'Sign up and create a key in seconds. One key unlocks every model on the platform.',
    icon: KeyRound,
  },
  {
    number: '02',
    title: 'Point your code',
    copy: 'Change the base URL in your existing Agent application. No other changes needed.',
    icon: Code2,
  },
  {
    number: '03',
    title: 'Scale with confidence',
    copy: 'Route across providers, monitor usage in real time, and optimize costs as you grow.',
    icon: Rocket,
  },
] as const

export function HowItWorks() {
  const { t } = useTranslation()
  return (
    <AnimateInView
      as='section'
      className='nexus-workflow border-t px-6 py-24 md:py-32'
    >
      <div className='mx-auto max-w-7xl'>
        <div className='mb-14 max-w-2xl'>
          <p className='nexus-section-kicker'>
            {t('Up and running in minutes')}
          </p>
          <h2 className='nexus-section-title mt-3'>
            {t('Three steps to every model.')}
          </h2>
        </div>
        <div className='nexus-workflow__track'>
          {steps.map(({ number, title, copy, icon: Icon }, index) => (
            <AnimateInView
              key={number}
              delay={index * 140}
              animation='fade-up'
              className='h-full'
            >
              <div className='nexus-workflow__step h-full'>
                <div className='nexus-workflow__node'>
                  <span>{number}</span>
                  <Icon className='size-5' />
                </div>
                <div className='nexus-workflow__copy'>
                  <p className='nexus-workflow__eyebrow'>
                    {t(`Step ${number}`)}
                  </p>
                  <h3>{t(title)}</h3>
                  <p>{t(copy)}</p>
                </div>
                {index < steps.length - 1 && (
                  <ArrowRight className='nexus-workflow__arrow hidden md:block' />
                )}
              </div>
            </AnimateInView>
          ))}
        </div>
      </div>
    </AnimateInView>
  )
}
