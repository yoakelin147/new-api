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
  CircleDot,
  Gauge,
  Route,
  WalletCards,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useStatus } from '@/hooks/use-status'
import { Button } from '@/components/ui/button'

const routeNodes = [
  { label: '稳定服务', value: '60%', tone: 'mint' },
  { label: '智能调度', value: '30%', tone: 'sky' },
  { label: '弹性保障', value: '10%', tone: 'lavender' },
] as const

/**
 * The landing hero for the relay product. It intentionally presents a
 * traffic-control narrative instead of the upstream project's generic model
 * catalog layout while keeping all existing CTA destinations intact.
 */
export function HeroCommandCenter() {
  const { t } = useTranslation()
  const { status } = useStatus()
  const docsUrl =
    (status?.docs_link as string | undefined) || 'https://docs.newapi.pro'

  const docsButton = docsUrl.startsWith('http') ? (
    <Button
      variant='outline'
      className='relay-hero__button relay-hero__button--secondary'
      render={<a href={docsUrl} target='_blank' rel='noopener noreferrer' />}
    >
      <BookOpen className='size-4' />
      {t('Docs')}
    </Button>
  ) : (
    <Button
      variant='outline'
      className='relay-hero__button relay-hero__button--secondary'
      render={<Link to={docsUrl} />}
    >
      <BookOpen className='size-4' />
      {t('Docs')}
    </Button>
  )

  return (
    <section className='relay-hero relative overflow-hidden px-6 pt-28 pb-20 md:pt-36 md:pb-28'>
      <div
        className='relay-hero__orb relay-hero__orb--sky'
        aria-hidden='true'
      />
      <div
        className='relay-hero__orb relay-hero__orb--mint'
        aria-hidden='true'
      />
      <div className='relay-hero__grid' aria-hidden='true' />

      <div className='relay-hero__content relative z-10 mx-auto max-w-7xl'>
        <div className='grid items-center gap-14 lg:grid-cols-[0.92fr_1.08fr] lg:gap-16'>
          <div className='max-w-2xl'>
            <div className='relay-hero__kicker landing-animate-fade-up mb-7 flex items-center gap-3 font-mono text-[10px] font-semibold tracking-[0.24em] uppercase opacity-0'>
              <span className='relay-hero__kicker-line h-px w-10' />
              {t('API Traffic Control')}
            </div>

            <div
              className='relay-hero__badge landing-animate-fade-up mb-6 inline-flex items-center gap-2 rounded-full border px-3.5 py-2 text-xs font-semibold opacity-0'
              style={{ animationDelay: '80ms' }}
            >
              <CircleDot className='size-3.5' />
              {t('AI Gateway Control Plane')}
            </div>

            <h1
              className='relay-hero__title landing-animate-fade-up text-[clamp(2.5rem,5vw,4rem)] leading-[1.04] font-black tracking-[-0.05em] opacity-0'
              style={{ animationDelay: '140ms' }}
            >
              {t('Unified API Gateway for')}
              <br />
              <span className='relay-hero__title-accent'>
                {t('Vast Range of AI Models')}
              </span>
            </h1>

            <div
              className='relay-hero__lead landing-animate-fade-up mt-7 max-w-xl text-base leading-8 opacity-0 md:text-lg'
              style={{ animationDelay: '220ms' }}
            >
              <p>
                {t(
                  '一个入口，连接所有模型供应商。自动路由、实时观测、按真实用量计费，让你的 AI 应用稳定地跑在正确的路径上。'
                )}
              </p>
            </div>

            <div
              className='landing-animate-fade-up mt-9 flex flex-wrap items-center gap-3 opacity-0'
              style={{ animationDelay: '300ms' }}
            >
              <Button
                className='relay-hero__button relay-hero__button--primary'
                render={
                  <Link
                    to='/dashboard/$section'
                    params={{ section: 'overview' }}
                  />
                }
              >
                {t('Get Started')}
                <ArrowRight className='size-4' />
              </Button>
              <Button
                variant='outline'
                className='relay-hero__button relay-hero__button--secondary'
                render={<Link to='/pricing' />}
              >
                {t('View Pricing')}
              </Button>
              {docsButton}
            </div>

            <p
              className='relay-hero__compatibility landing-animate-fade-up mt-6 text-xs opacity-0'
              style={{ animationDelay: '360ms' }}
            >
              {t('No client migration required')} ·{' '}
              {t('Compatible API surfaces')}: OpenAI / Responses / Claude /
              Gemini
            </p>
          </div>

          <div
            className='relay-hero__panel-wrap landing-animate-fade-up opacity-0'
            style={{ animationDelay: '260ms' }}
          >
            <div className='relay-hero__panel-header mb-3 flex items-center justify-between px-1 font-mono text-[10px] font-semibold tracking-[0.16em] uppercase'>
              <span>{t('Service experience')}</span>
              <span className='flex items-center gap-2'>
                <span className='relay-hero__live-dot' /> {t('Live')}
              </span>
            </div>

            <div className='relay-hero__traffic-panel'>
              <div className='flex items-start justify-between gap-4'>
                <div>
                  <p className='relay-hero__panel-label'>
                    {t('One simple connection')}
                  </p>
                  <p className='relay-hero__endpoint text-xs'>
                    {t('Everything you need, ready to use')}
                  </p>
                </div>
                <div className='relay-hero__normal-state'>
                  <Check className='size-3.5' /> {t('All systems normal')}
                </div>
              </div>

              <div className='relay-hero__flow mt-8'>
                <div className='relay-hero__request-node'>
                  <span className='relay-hero__node-caption'>
                    {t('Your application')}
                  </span>
                  <strong>{t('Your app')}</strong>
                  <span className='relay-hero__node-port'>
                    {t('Ready to connect')}
                  </span>
                </div>
                <div
                  className='relay-hero__route-lane relay-hero__route-lane--one'
                  aria-hidden='true'
                >
                  <span className='relay-hero__packet' />
                </div>
                <div className='relay-hero__router-node'>
                  <Route className='size-5' />
                  <span>{t('Smart matching')}</span>
                  <strong>{t('Best experience')}</strong>
                </div>
                <div
                  className='relay-hero__route-lane relay-hero__route-lane--two'
                  aria-hidden='true'
                >
                  <span className='relay-hero__packet' />
                </div>
                <div className='relay-hero__provider-node'>
                  <span className='relay-hero__node-caption'>
                    {t('AI models')}
                  </span>
                  <strong>{t('The right model')}</strong>
                  <span className='relay-hero__node-port'>
                    {t('Reliable and ready')}
                  </span>
                </div>
              </div>

              <div className='mt-10'>
                <div className='mb-3 flex items-center justify-between'>
                  <span className='relay-hero__panel-label'>
                    {t('A stable experience, every time')}
                  </span>
                  <span className='relay-hero__panel-label'>100%</span>
                </div>
                <div className='relay-hero__traffic-bar'>
                  <span className='relay-hero__traffic-bar-segment relay-hero__traffic-bar-segment--mint' />
                  <span className='relay-hero__traffic-bar-segment relay-hero__traffic-bar-segment--sky' />
                  <span className='relay-hero__traffic-bar-segment relay-hero__traffic-bar-segment--lavender' />
                </div>
                <div className='mt-3 grid grid-cols-3 gap-2'>
                  {routeNodes.map((node) => (
                    <div key={node.label} className='relay-hero__route-stat'>
                      <span
                        className={`relay-hero__route-dot relay-hero__route-dot--${node.tone}`}
                      />
                      <span>{t(node.label)}</span>
                      <strong>{node.value}</strong>
                    </div>
                  ))}
                </div>
              </div>
            </div>

            <div className='relay-hero__telemetry mt-3 grid grid-cols-3 gap-2'>
              <div>
                <Gauge className='size-3.5' />
                <span>{t('Fast response')}</span>
                <strong>142ms</strong>
              </div>
              <div>
                <Check className='size-3.5' />
                <span>{t('Reliable service')}</span>
                <strong>99.98%</strong>
              </div>
              <div>
                <WalletCards className='size-3.5' />
                <span>{t('Transparent pricing')}</span>
                <strong>100%</strong>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
