import { useState } from 'react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { cn } from '@/lib/utils'

import { CompletedMockup } from './completed-mockup'
import type { ControlSize, MediaTheme } from './controls-types'
import { gapForSize } from './controls-utils'
import { DefaultMockup } from './default-mockup'
import { ErrorMockup } from './error-mockup'
import { ProgressMockup } from './progress-mockup'
import { SearchingMockup } from './searching-mockup'
import { useDemoRow } from './use-demo-row'

export function InteractiveStatePreview() {
  const [theme, setTheme] = useState<MediaTheme>('movie')

  return (
    <Card>
      <CardHeader>
        <CardTitle>Interactive State Preview</CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="flex items-center gap-4">
          <div className="flex gap-1">
            <Button
              variant={theme === 'movie' ? 'default' : 'outline'}
              size="sm"
              onClick={() => setTheme('movie')}
            >
              Movie
            </Button>
            <Button
              variant={theme === 'tv' ? 'default' : 'outline'}
              size="sm"
              onClick={() => setTheme('tv')}
            >
              TV
            </Button>
          </div>
          <p className="text-muted-foreground text-sm">
            Click buttons to trigger state transitions
          </p>
        </div>

        <div className="space-y-4">
          {(['lg', 'sm', 'xs'] as const).map((size) => (
            <DemoRow key={`${theme}-${size}`} theme={theme} size={size} />
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

function DemoRow({ theme, size }: { theme: MediaTheme; size: ControlSize }) {
  const { state, monitored, setMonitored, runAutoSearch, runManualSearch, isDefault } = useDemoRow()
  const gap = gapForSize(size)

  const content = (() => {
    switch (state.type) {
      case 'searching': {
        return <SearchingMockup theme={theme} size={size} mode={state.mode} fullWidth />
      }
      case 'progress': {
        return (
          <ProgressMockup
            theme={theme}
            size={size}
            progress={state.percent}
            paused={false}
            fullWidth
          />
        )
      }
      case 'completed': {
        return <CompletedMockup theme={theme} size={size} fullWidth />
      }
      case 'error': {
        return <ErrorMockup theme={theme} size={size} message={state.message} fullWidth />
      }
      default: {
        return null
      }
    }
  })()

  return (
    <div className="flex items-center gap-4">
      <span className="text-muted-foreground w-6 text-xs font-medium">{size}</span>
      <div className="relative flex-1">
        <div className={cn('flex items-center', gap, !isDefault && 'invisible')}>
          <DefaultMockup
            theme={theme}
            size={size}
            monitored={monitored}
            onManualSearch={runManualSearch}
            onAutoSearch={runAutoSearch}
            onMonitoredChange={setMonitored}
          />
        </div>
        {!isDefault && <div className="absolute inset-0 flex items-center">{content}</div>}
      </div>
    </div>
  )
}
