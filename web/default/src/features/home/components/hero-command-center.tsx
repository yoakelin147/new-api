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
import { Link } from '@tanstack/react-router'
import {
  ArrowRight,
  BookOpen,
  Check,
  Gauge,
  Route,
  Sparkles,
  WalletCards,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useCountUp } from '@/hooks/use-count-up'
import { useSpotlight } from '@/hooks/use-spotlight'
import { useStatus } from '@/hooks/use-status'
import { Button } from '@/components/ui/button'

const routeNodes = [
  { label: '稳定服务', value: '60%', tone: 'mint' },
  { label: '智能调度', value: '30%', tone: 'sky' },
  { label: '弹性保障', value: '10%', tone: 'lavender' },
] as const

function TelemetryChip(props: {
  icon: React.ReactNode
  label: string
  value: string
  numericValue?: number
  suffix?: string
  decimals?: number
}) {
  const { ref, display } = useCountUp<HTMLDivElement>(
    props.numericValue ?? Number.NaN,
    { decimals: props.decimals ?? 0, delay: 600 }
  )
  return (
    <div ref={ref}>
      {props.icon}
      <span>{props.label}</span>
      <strong>
        {props.numericValue !== undefined
          ? `${display}${props.suffix ?? ''}`
          : props.value}
      </strong>
    </div>
  )
}

/**
 * Landing hero: left copy + right traffic-flow panel showing a request
 * moving from the user's app through the router to a model provider.
 */
export function HeroCommandCenter() {
  const { t } = useTranslation()
  const { status } = useStatus()
  const panelRef = useSpotlight<HTMLDivElement>()
  const docsUrl =
    (status?.docs_link as string | undefined) || 'https://docs.newapi.pro'

  const docsButton = docsUrl.startsWith('http') ? (
    <Button
      variant='outline'
      className='nexus-hero__button nexus-hero__button--secondary'
      render={<a href={docsUrl} target='_blank' rel='noopener noreferrer' />}
    >
      <BookOpen className='size-4' />
      {t('Docs')}
    </Button>
  ) : (
    <Button
      variant='outline'
      className='nexus-hero__button nexus-hero__button--secondary'
      render={<Link to={docsUrl} />}
    >
      <BookOpen className='size-4' />
      {t('Docs')}
    </Button>
  )

  return (
    <section className='nexus-hero relative overflow-hidden px-6 pt-20 pb-16 md:pt-24 md:pb-20'>
      <div className='nexus-hero__orb nexus-hero__orb--a' aria-hidden='true' />
      <div className='nexus-hero__orb nexus-hero__orb--b' aria-hidden='true' />
      <div className='nexus-hero__grid' aria-hidden='true' />

      <div className='nexus-hero__content relative z-10 mx-auto max-w-7xl'>
        <div className='grid items-center gap-14 lg:grid-cols-[0.92fr_1.08fr] lg:gap-16'>
          <div className='max-w-2xl'>
            <div className='nexus-hero__kicker landing-animate-fade-up mb-7 flex items-center gap-3 font-mono text-xs font-semibold tracking-[0.24em] uppercase opacity-0'>
              <span className='nexus-hero__kicker-line h-px w-10' />
              {t('AI Model Aggregation Platform')}
            </div>

            <div
              className='nexus-hero__badge landing-animate-fade-up mb-6 inline-flex items-center gap-2 rounded-full border px-3.5 py-2 text-xs font-semibold opacity-0'
              style={{ animationDelay: '80ms' }}
            >
              <Sparkles className='size-3.5' />
              {t('One API for every frontier model')}
            </div>

            <h1
              className='nexus-hero__title landing-animate-fade-up text-[clamp(2.5rem,5vw,4rem)] leading-[1.04] font-black tracking-[-0.05em] opacity-0'
              style={{ animationDelay: '140ms' }}
            >
              {t('接入前沿优质模型')}
              <br />
              <span className='nexus-hero__title-accent'>
                {t('通过统一聚合网关')}
              </span>
            </h1>

            <div
              className='nexus-hero__lead landing-animate-fade-up mt-7 max-w-xl text-base leading-8 opacity-0 md:text-lg'
              style={{ animationDelay: '220ms' }}
            >
              <p>
                {t(
                  'Connect to leading model providers through a single unified API. Intelligent routing, real-time observability, and pay-as-you-go billing built for production workloads.'
                )}
              </p>
            </div>

            <div
              className='landing-animate-fade-up mt-9 flex flex-wrap items-center gap-3 opacity-0'
              style={{ animationDelay: '300ms' }}
            >
              <Button
                className='nexus-hero__button nexus-hero__button--primary'
                render={
                  <Link
                    to='/dashboard/$section'
                    params={{ section: 'overview' }}
                  />
                }
              >
                {t('Start Building')}
                <ArrowRight className='size-4' />
              </Button>
              <Button
                variant='outline'
                className='nexus-hero__button nexus-hero__button--secondary'
                render={<Link to='/pricing' />}
              >
                {t('View Pricing')}
              </Button>
              {docsButton}
            </div>
          </div>

          <div
            className='nexus-hero__panel-wrap landing-animate-fade-up opacity-0'
            style={{ animationDelay: '260ms' }}
          >
            <div className='nexus-hero__panel-header mb-3 flex items-center justify-between px-1 font-mono text-xs font-semibold tracking-[0.16em] uppercase'>
              <span>{t('Service experience')}</span>
              <span className='flex items-center gap-2'>
                <span className='nexus-hero__live-dot' /> {t('Live')}
              </span>
            </div>

            <div
              ref={panelRef}
              className='nexus-hero__traffic-panel relay-spotlight'
            >
              <div className='flex items-start justify-between gap-4'>
                <div>
                  <p className='nexus-hero__panel-label'>
                    {t('One simple connection')}
                  </p>
                  <p className='nexus-hero__endpoint text-sm'>
                    {t('Everything you need, ready to use')}
                  </p>
                </div>
                <div className='nexus-hero__normal-state'>
                  <Check className='size-3.5' /> {t('All systems normal')}
                </div>
              </div>

              <div className='nexus-hero__flow mt-8'>
                <div className='nexus-hero__request-node'>
                  <span className='nexus-hero__node-caption'>
                    {t('Your application')}
                  </span>
                  <strong>{t('Your app')}</strong>
                  <span className='nexus-hero__node-port'>
                    {t('Ready to connect')}
                  </span>
                </div>
                <div
                  className='nexus-hero__route-lane nexus-hero__route-lane--one'
                  aria-hidden='true'
                >
                  <span className='nexus-hero__packet' />
                </div>
                <div className='nexus-hero__router-node'>
                  <Route className='size-5' />
                  <span>{t('Smart matching')}</span>
                  <strong>{t('Best experience')}</strong>
                </div>
                <div
                  className='nexus-hero__route-lane nexus-hero__route-lane--two'
                  aria-hidden='true'
                >
                  <span className='nexus-hero__packet' />
                </div>
                <div className='nexus-hero__provider-node'>
                  <span className='nexus-hero__node-caption'>
                    {t('AI models')}
                  </span>
                  <strong>{t('The right model')}</strong>
                  <span className='nexus-hero__node-port'>
                    {t('Reliable and ready')}
                  </span>
                </div>
              </div>

              <div className='mt-10'>
                <div className='mb-3 flex items-center justify-between'>
                  <span className='nexus-hero__panel-label'>
                    {t('A stable experience, every time')}
                  </span>
                  <span className='nexus-hero__panel-label'>100%</span>
                </div>
                <div className='nexus-hero__traffic-bar'>
                  <span className='nexus-hero__traffic-bar-segment nexus-hero__traffic-bar-segment--mint' />
                  <span className='nexus-hero__traffic-bar-segment nexus-hero__traffic-bar-segment--sky' />
                  <span className='nexus-hero__traffic-bar-segment nexus-hero__traffic-bar-segment--lavender' />
                </div>
                <div className='mt-3 grid grid-cols-3 gap-2'>
                  {routeNodes.map((node) => (
                    <div key={node.label} className='nexus-hero__route-stat'>
                      <span
                        className={`nexus-hero__route-dot nexus-hero__route-dot--${node.tone}`}
                      />
                      <span>{t(node.label)}</span>
                      <strong>{node.value}</strong>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            <div className='nexus-hero__telemetry mt-3 grid grid-cols-3 gap-2'>
              <TelemetryChip
                icon={<Gauge className='size-3.5' />}
                label={t('Fast response')}
                value='90ms'
                numericValue={90}
                suffix='ms'
              />
              <TelemetryChip
                icon={<Check className='size-3.5' />}
                label={t('Reliable service')}
                value='99.98%'
                numericValue={99.98}
                suffix='%'
                decimals={2}
              />
              <TelemetryChip
                icon={<WalletCards className='size-3.5' />}
                label={t('Transparent pricing')}
                value='100%'
                numericValue={100}
                suffix='%'
              />
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
