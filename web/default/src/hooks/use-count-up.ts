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
import { useEffect, useRef, useState } from 'react'

interface UseCountUpOptions {
  /** Animation duration in ms. Default 1400. */
  duration?: number
  /** Delay before starting in ms. Default 0. */
  delay?: number
  /** Decimal places to keep. Default 0. */
  decimals?: number
  /** Only start when the returned ref enters the viewport. Default true. */
  startOnView?: boolean
}

function easeOutCubic(progress: number): number {
  return 1 - Math.pow(1 - progress, 3)
}

/**
 * Animates a number from 0 to `target` with an ease-out curve.
 * Returns the ref to attach to the element and the formatted display value.
 * Respects `prefers-reduced-motion` by jumping straight to the target.
 */
export function useCountUp<T extends HTMLElement = HTMLDivElement>(
  target: number,
  options?: UseCountUpOptions
) {
  const {
    duration = 1400,
    delay = 0,
    decimals = 0,
    startOnView = true,
  } = options ?? {}

  const ref = useRef<T>(null)
  const [display, setDisplay] = useState<string | null>(null)

  useEffect(() => {
    const mq = window.matchMedia('(prefers-reduced-motion: reduce)')
    if (mq.matches || !Number.isFinite(target)) {
      // Defer to the next frame so the effect body stays side-effect-only
      // (DOM subscription) instead of synchronously cascading a render.
      const id = requestAnimationFrame(() =>
        setDisplay(target.toFixed(decimals))
      )
      return () => cancelAnimationFrame(id)
    }

    let rafId = 0
    let timerId = 0
    let started = false

    const run = () => {
      if (started) return
      started = true
      const startTime = performance.now() + delay
      const tick = (now: number) => {
        const elapsed = now - startTime
        if (elapsed < 0) {
          rafId = requestAnimationFrame(tick)
          return
        }
        const progress = Math.min(elapsed / duration, 1)
        setDisplay((target * easeOutCubic(progress)).toFixed(decimals))
        if (progress < 1) rafId = requestAnimationFrame(tick)
      }
      rafId = requestAnimationFrame(tick)
    }

    if (!startOnView || !ref.current) {
      run()
    } else {
      const observer = new IntersectionObserver(
        ([entry]) => {
          if (entry.isIntersecting) {
            run()
            observer.disconnect()
          }
        },
        { threshold: 0.4 }
      )
      observer.observe(ref.current)
      timerId = window.setTimeout(run, 2500) // safety net
      return () => {
        observer.disconnect()
        window.clearTimeout(timerId)
        cancelAnimationFrame(rafId)
      }
    }

    return () => {
      window.clearTimeout(timerId)
      cancelAnimationFrame(rafId)
    }
  }, [target, duration, delay, decimals, startOnView])

  return { ref, display: display ?? (0).toFixed(decimals) }
}
