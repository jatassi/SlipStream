import { PageHeader } from '@/components/layout/page-header'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

import { CompletedFlashDemo } from './completed-flash-demo'
import { InteractiveStatePreview } from './interactive-state-preview'
import { LiveDefaultRow } from './live-default-row'
import { MonitorMockup } from './monitor-mockup'
import { ProgressMockup } from './progress-mockup'
import { StateRow } from './state-row'

export function ControlsShowcasePage() {
  return (
    <div className="space-y-8">
      <PageHeader
        title="MediaSearchMonitorControls Showcase"
        description="Every permutation of size, theme, state, and monitored flag"
      />
      <LiveComponentsSection />
      <InteractiveStatePreview />
      <VisualStatesSection />
      <ChasingLightsSection />
      <ProgressBarSection />
      <CompletionFlashSection />
      <MonitorButtonSection />
    </div>
  )
}

function LiveComponentsSection() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Live Components — Default State</CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        <p className="text-muted-foreground text-sm">
          These are real <code>MediaSearchMonitorControls</code> instances. Click buttons to
          trigger state transitions.
        </p>
        <div className="grid grid-cols-1 gap-6">
          {(['movie', 'tv'] as const).map((theme) => (
            <div key={theme} className="space-y-4">
              <h3 className="text-muted-foreground text-sm font-semibold tracking-wider uppercase">
                {theme === 'movie' ? 'Movie Theme (Orange)' : 'TV Theme (Blue)'}
              </h3>
              <div className="space-y-3">
                {(['lg', 'sm', 'xs'] as const).map((size) => (
                  <LiveDefaultRow key={size} theme={theme} size={size} />
                ))}
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

function VisualStatesSection() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>All Visual States</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-muted-foreground mb-6 text-sm">
          Static mockups of every state x size x theme combination.
        </p>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b">
                <th className="w-32 px-3 py-2 text-left">State</th>
                <th className="px-3 py-2 text-center">Movie / lg</th>
                <th className="px-3 py-2 text-center">Movie / sm</th>
                <th className="px-3 py-2 text-center">Movie / xs</th>
                <th className="px-3 py-2 text-center">TV / lg</th>
                <th className="px-3 py-2 text-center">TV / sm</th>
                <th className="px-3 py-2 text-center">TV / xs</th>
              </tr>
            </thead>
            <tbody>
              <StateRow label="Default (Monitored)" state="default-monitored" />
              <StateRow label="Default (Unmonitored)" state="default-unmonitored" />
              <StateRow label="Searching (Auto)" state="searching-auto" />
              <StateRow label="Searching (Manual)" state="searching-manual" />
              <StateRow label="Progress (35%)" state="progress-35" />
              <StateRow label="Progress (72%)" state="progress-72" />
              <StateRow label="Progress (Paused)" state="progress-paused" />
              <StateRow label="Completed" state="completed" />
              <StateRow label="Error (Not Found)" state="error-notfound" />
              <StateRow label="Error (Failed)" state="error-failed" />
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  )
}

function ChasingLightsSection() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Chasing Lights Animation</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-muted-foreground text-sm">
          The chasing lights effect used during auto-search at lg and sm sizes.
        </p>
        <div className="flex flex-wrap items-center gap-6">
          <ChasingLightDemo label="Movie / lg" className="chasing-lights-movie" />
          <ChasingLightDemo label="Movie / sm" className="chasing-lights-movie" sm />
          <ChasingLightDemo label="TV / lg" className="chasing-lights-tv" />
          <ChasingLightDemo label="TV / sm" className="chasing-lights-tv" sm />
        </div>
      </CardContent>
    </Card>
  )
}

function ChasingLightDemo({ label, className, sm }: { label: string; className: string; sm?: boolean }) {
  return (
    <div className="space-y-2">
      <Badge variant="outline">{label}</Badge>
      <div className={className}>
        <div className="bg-card absolute inset-0 z-[1] rounded-md" />
        <Button
          variant="outline"
          size={sm ? 'sm' : 'default'}
          disabled
          className={sm ? 'relative z-[2] h-8 text-xs' : 'relative z-[2]'}
        >
          Searching...
        </Button>
      </div>
    </div>
  )
}

function ProgressBarSection() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Progress Bar Variations</CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        <p className="text-muted-foreground text-sm">
          Progress state at various percentages with shimmer, edge glow, and inset glow.
        </p>
        {(['movie', 'tv'] as const).map((theme) => (
          <ProgressThemeBlock key={theme} theme={theme} />
        ))}
      </CardContent>
    </Card>
  )
}

function ProgressThemeBlock({ theme }: { theme: 'movie' | 'tv' }) {
  return (
    <div className="space-y-3">
      <h3 className="text-muted-foreground text-sm font-semibold tracking-wider uppercase">
        {theme === 'movie' ? 'Movie' : 'TV'}
      </h3>
      <div className="space-y-2">
        {[5, 25, 50, 75, 95].map((pct) => (
          <div key={pct} className="flex items-center gap-4">
            <span className="text-muted-foreground w-8 text-right text-xs">{pct}%</span>
            <div className="flex items-center gap-3">
              <ProgressMockup theme={theme} size="lg" progress={pct} paused={false} />
              <ProgressMockup theme={theme} size="sm" progress={pct} paused={false} />
              <ProgressMockup theme={theme} size="xs" progress={pct} paused={false} />
            </div>
          </div>
        ))}
        <div className="flex items-center gap-4">
          <span className="text-muted-foreground w-8 text-right text-xs">50%</span>
          <div className="flex items-center gap-3">
            {(['lg', 'sm', 'xs'] as const).map((size) => (
              <div key={size} className="space-y-0.5">
                <ProgressMockup theme={theme} size={size} progress={50} paused />
                <span className="text-[10px] text-amber-400">paused</span>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}

function CompletionFlashSection() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Completion Flash</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-muted-foreground text-sm">
          The flash animation plays once on completion. Click to restart.
        </p>
        <div className="flex flex-wrap items-start gap-6">
          <CompletedFlashDemo theme="movie" />
          <CompletedFlashDemo theme="tv" />
        </div>
      </CardContent>
    </Card>
  )
}

function MonitorButtonSection() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Monitor Button Styles</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-muted-foreground text-sm">
          Monitor/Unmonitor button appearance across all sizes and themes.
        </p>
        <div className="grid grid-cols-2 gap-6">
          {(['movie', 'tv'] as const).map((theme) => (
            <div key={theme} className="space-y-3">
              <h3 className="text-muted-foreground text-sm font-semibold tracking-wider uppercase">
                {theme === 'movie' ? 'Movie' : 'TV'}
              </h3>
              <div className="space-y-2">
                {(['lg', 'sm', 'xs'] as const).map((size) => (
                  <div key={size} className="flex items-center gap-4">
                    <span className="text-muted-foreground w-6 text-xs">{size}</span>
                    <MonitorMockup theme={theme} size={size} monitored />
                    <MonitorMockup theme={theme} size={size} monitored={false} />
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}
