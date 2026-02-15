import type { ReactNode } from 'react'

import { cn } from '@/lib/utils'

import { CompletedMockup } from './completed-mockup'
import type { ControlSize, MediaTheme, StateName } from './controls-types'
import { gapForSize } from './controls-utils'
import { DefaultMockup } from './default-mockup'
import { ErrorMockup } from './error-mockup'
import { ProgressMockup } from './progress-mockup'
import { SearchingMockup } from './searching-mockup'

export function StateRow({ label, state }: { label: string; state: StateName }) {
  return (
    <tr className="border-b last:border-0">
      <td className="px-3 py-3 text-xs font-medium">{label}</td>
      {(['movie', 'tv'] as const).map((theme) =>
        (['lg', 'sm', 'xs'] as const).map((size) => (
          <td key={`${theme}-${size}`} className="px-3 py-3">
            <div className="flex justify-center">
              <StateMockup state={state} theme={theme} size={size} />
            </div>
          </td>
        )),
      )}
    </tr>
  )
}

function renderState(props: { state: StateName; theme: MediaTheme; size: ControlSize }, fw: boolean): ReactNode {
  const { state, theme, size } = props
  const stateMap: Record<StateName, ReactNode> = {
    'default-monitored': <DefaultMockup theme={theme} size={size} monitored />,
    'default-unmonitored': <DefaultMockup theme={theme} size={size} monitored={false} />,
    'searching-auto': <SearchingMockup theme={theme} size={size} mode="auto" fullWidth={fw} />,
    'searching-manual': <SearchingMockup theme={theme} size={size} mode="manual" fullWidth={fw} />,
    'progress-35': <ProgressMockup theme={theme} size={size} progress={35} paused={false} fullWidth={fw} />,
    'progress-72': <ProgressMockup theme={theme} size={size} progress={72} paused={false} fullWidth={fw} />,
    'progress-paused': <ProgressMockup theme={theme} size={size} progress={50} paused fullWidth={fw} />,
    'completed': <CompletedMockup theme={theme} size={size} fullWidth={fw} />,
    'error-notfound': <ErrorMockup theme={theme} size={size} message="Not Found" fullWidth={fw} />,
    'error-failed': <ErrorMockup theme={theme} size={size} message="Failed" fullWidth={fw} />,
  }
  return stateMap[state]
}

function StateMockup({ state, theme, size }: { state: StateName; theme: MediaTheme; size: ControlSize }) {
  const isDefault = state === 'default-monitored' || state === 'default-unmonitored'
  const content = renderState({ state, theme, size }, !isDefault)

  if (isDefault) {
    return content
  }

  return (
    <div className="relative">
      <div className={cn('invisible flex items-center', gapForSize(size))}>
        <DefaultMockup theme={theme} size={size} monitored />
      </div>
      <div className="absolute inset-0 flex items-center">{content}</div>
    </div>
  )
}
