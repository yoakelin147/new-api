/* Copyright (C) 2023-2026 QuantumNous */
import { ArrowRight, Check, Settings2, Waypoints, Workflow } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { AnimateInView } from '@/components/animate-in-view'

const steps = [
  {
    number: '01',
    title: 'Choose what you need',
    copy: 'Pick a model or service that fits your task and start right away.',
    icon: Settings2,
  },
  {
    number: '02',
    title: 'Start creating',
    copy: 'Use the same simple experience for conversations, code, images, and more.',
    icon: Waypoints,
  },
  {
    number: '03',
    title: 'Keep moving',
    copy: 'Rely on a stable service with clear usage and pricing as your work grows.',
    icon: Workflow,
  },
] as const

export function HowItWorks() {
  const { t } = useTranslation()
  return (
    <AnimateInView
      as='section'
      className='relay-workflow border-t px-6 py-24 md:py-32'
    >
      <div className='mx-auto max-w-7xl'>
        <div className='mb-14 max-w-2xl'>
          <p className='relay-section-kicker'>{t('A simple way to use AI')}</p>
          <h2 className='relay-section-title mt-3'>
            {t('From your idea to the result.')}
          </h2>
        </div>
        <div className='relay-workflow__track'>
          {steps.map(({ number, title, copy, icon: Icon }, index) => (
            <div key={number} className='relay-workflow__step'>
              <div className='relay-workflow__node'>
                <span>{number}</span>
                <Icon className='size-5' />
              </div>
              <div className='relay-workflow__copy'>
                <p className='relay-workflow__eyebrow'>{t(`Step ${number}`)}</p>
                <h3>{t(title)}</h3>
                <p>{t(copy)}</p>
              </div>
              {index < steps.length - 1 && (
                <ArrowRight className='relay-workflow__arrow hidden md:block' />
              )}
            </div>
          ))}
        </div>
        <div className='relay-workflow__footer mt-12 flex flex-wrap items-center gap-3 text-xs'>
          <Check className='size-4' />
          <span>{t('No client migration required')}</span>
          <span className='relay-workflow__separator'>/</span>
          <span>{t('Compatible API surfaces')}</span>
        </div>
      </div>
    </AnimateInView>
  )
}
