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
  Boxes,
  Gauge,
  Layers,
  Sparkles,
  Zap,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useCountUp } from '@/hooks/use-count-up'
import { useSpotlight } from '@/hooks/use-spotlight'
import { useStatus } from '@/hooks/use-status'
import { Button } from '@/components/ui/button'

const modelChips = [
  { name: 'GPT', tone: 'mint', x: '8%', y: '12%', delay: '0s' },
  { name: 'Claude', tone: 'sky', x: '62%', y: '6%', delay: '0.6s' },
  { name: 'Gemini', tone: 'violet', x: '76%', y: '38%', delay: '1.2s' },
  { name: 'DeepSeek', tone: 'mint', x: '12%', y: '56%', delay: '0.3s' },
  { name: 'Llama', tone: 'sky', x: '55%', y: '72%', delay: '0.9s' },
  { name: 'Qwen', tone: 'violet', x: '30%', y: '84%', delay: '1.5s' },
  { name: 'Midjourney', tone: 'mint', x: '72%', y: '82%', delay: '0.4s' },
  { name: 'Sora', tone: 'sky', x: '4%', y: '82%', delay: '1.1s' },
] as const

function MetricChip(props: {
  icon: React.ReactNode
  label: string
  numericValue: number
  suffix: string
  decimals?: number
}) {
  const { ref, display } = useCountUp<HTMLDivElement>(props.numericValue, {
    decimals: props.decimals ?? 0,
    delay: 500,
  })
  return (
    <div ref={ref} className='nexus-hero__metric'>
      {props.icon}
      <span>{props.label}</span>
      <strong>
        {display}
        {props.suffix}
      </strong>
    </div>
  )
}

/**
 * Landing hero for the model-aggregation narrative: a single unified entry
 * point surrounded by a constellation of model/provider chips.
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
        <div className='grid items-center gap-14 lg:grid-cols-[1fr_1.05fr] lg:gap-16'>
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
              {t('Access any frontier model')}
              <br />
              <span className='nexus-hero__title-accent'>
                {t('through one unified gateway')}
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
            <div ref={panelRef} className='nexus-hero__panel relay-spotlight'>
              <div className='nexus-hero__panel-header'>
                <span>{t('Unified model hub')}</span>
                <span className='nexus-hero__live'>
                  <span className='nexus-hero__live-dot' /> {t('Live')}
                </span>
              </div>

              <div className='nexus-hero__constellation'>
                <svg
                  className='nexus-hero__links'
                  viewBox='0 0 400 300'
                  aria-hidden='true'
                >
                  <line x1='200' y1='150' x2='60' y2='45' />
                  <line x1='200' y1='150' x2='330' y2='30' />
                  <line x1='200' y1='150' x2='350' y2='130' />
                  <line x1='200' y1='150' x2='55' y2='185' />
                  <line x1='200' y1='150' x2='295' y2='230' />
                  <line x1='200' y1='150' x2='140' y2='265' />
                  <line x1='200' y1='150' x2='330' y2='260' />
                  <line x1='200' y1='150' x2='25' y2='260' />
                </svg>
                <div className='nexus-hero__hub'>
                  <Layers className='size-5' />
                  <span>{t('Unified API')}</span>
                </div>
                {modelChips.map((chip) => (
                  <div
                    key={chip.name}
                    className={`nexus-hero__chip nexus-hero__chip--${chip.tone}`}
                    style={{
                      left: chip.x,
                      top: chip.y,
                      animationDelay: chip.delay,
                    }}
                  >
                    {chip.name}
                  </div>
                ))}
              </div>

              <div className='nexus-hero__metrics'>
                <MetricChip
                  icon={<Boxes className='size-3.5' />}
                  label={t('Models')}
                  numericValue={300}
                  suffix='+'
                />
                <MetricChip
                  icon={<Zap className='size-3.5' />}
                  label={t('Latency p95')}
                  numericValue={90}
                  suffix='ms'
                />
                <MetricChip
                  icon={<Gauge className='size-3.5' />}
                  label={t('Uptime')}
                  numericValue={99.98}
                  suffix='%'
                  decimals={2}
                />
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
