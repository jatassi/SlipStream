import { useState, useRef, useCallback, useEffect } from 'react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { MediaSearchMonitorControls } from '@/components/search'
import {
  UserSearch,
  Zap,
  Eye,
  EyeOff,
  AlertCircle,
  Check,
  Download,
} from 'lucide-react'
import { cn } from '@/lib/utils'

type MediaTheme = 'movie' | 'tv'
type ControlSize = 'lg' | 'sm' | 'xs'

// ---------------------------------------------------------------------------
// Showcase page
// ---------------------------------------------------------------------------

export function ControlsShowcasePage() {
  return (
    <div className="space-y-8">
      <PageHeader
        title="MediaSearchMonitorControls Showcase"
        description="Every permutation of size, theme, state, and monitored flag"
      />

      {/* Section 1: Live components in default state */}
      <Card>
        <CardHeader>
          <CardTitle>Live Components — Default State</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <p className="text-sm text-muted-foreground">
            These are real <code>MediaSearchMonitorControls</code> instances. Click buttons to trigger state transitions.
          </p>

          <div className="grid grid-cols-1 gap-6">
            {(['movie', 'tv'] as const).map((theme) => (
              <div key={theme} className="space-y-4">
                <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
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

      {/* Section 1b: Interactive state preview */}
      <InteractiveStatePreview />

      {/* Section 2: Visual states grid */}
      <Card>
        <CardHeader>
          <CardTitle>All Visual States</CardTitle>
        </CardHeader>
        <CardContent>
          <p className="text-sm text-muted-foreground mb-6">
            Static mockups of every state × size × theme combination.
          </p>

          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b">
                  <th className="text-left py-2 px-3 w-32">State</th>
                  <th className="text-center py-2 px-3">Movie / lg</th>
                  <th className="text-center py-2 px-3">Movie / sm</th>
                  <th className="text-center py-2 px-3">Movie / xs</th>
                  <th className="text-center py-2 px-3">TV / lg</th>
                  <th className="text-center py-2 px-3">TV / sm</th>
                  <th className="text-center py-2 px-3">TV / xs</th>
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

      {/* Section 3: Chasing lights animation */}
      <Card>
        <CardHeader>
          <CardTitle>Chasing Lights Animation</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            The chasing lights effect used during auto-search at lg and sm sizes.
          </p>
          <div className="flex flex-wrap gap-6 items-center">
            <div className="space-y-2">
              <Badge variant="outline">Movie / lg</Badge>
              <div className="chasing-lights-movie">
                <div className="absolute inset-0 rounded-md bg-card z-[1]" />
                <Button variant="outline" disabled className="relative z-[2]">
                  Searching...
                </Button>
              </div>
            </div>
            <div className="space-y-2">
              <Badge variant="outline">Movie / sm</Badge>
              <div className="chasing-lights-movie">
                <div className="absolute inset-0 rounded-md bg-card z-[1]" />
                <Button variant="outline" size="sm" disabled className="relative z-[2] h-8 text-xs">
                  Searching...
                </Button>
              </div>
            </div>
            <div className="space-y-2">
              <Badge variant="outline">TV / lg</Badge>
              <div className="chasing-lights-tv">
                <div className="absolute inset-0 rounded-md bg-card z-[1]" />
                <Button variant="outline" disabled className="relative z-[2]">
                  Searching...
                </Button>
              </div>
            </div>
            <div className="space-y-2">
              <Badge variant="outline">TV / sm</Badge>
              <div className="chasing-lights-tv">
                <div className="absolute inset-0 rounded-md bg-card z-[1]" />
                <Button variant="outline" size="sm" disabled className="relative z-[2] h-8 text-xs">
                  Searching...
                </Button>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Section 4: Progress bar variations */}
      <Card>
        <CardHeader>
          <CardTitle>Progress Bar Variations</CardTitle>
        </CardHeader>
        <CardContent className="space-y-6">
          <p className="text-sm text-muted-foreground">
            Progress state at various percentages with shimmer, edge glow, and inset glow.
          </p>

          {(['movie', 'tv'] as const).map((theme) => (
            <div key={theme} className="space-y-3">
              <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
                {theme === 'movie' ? 'Movie' : 'TV'}
              </h3>
              <div className="space-y-2">
                {[5, 25, 50, 75, 95].map((pct) => (
                  <div key={pct} className="flex items-center gap-4">
                    <span className="text-xs text-muted-foreground w-8 text-right">{pct}%</span>
                    <div className="flex items-center gap-3">
                      <ProgressMockup theme={theme} size="lg" progress={pct} paused={false} />
                      <ProgressMockup theme={theme} size="sm" progress={pct} paused={false} />
                      <ProgressMockup theme={theme} size="xs" progress={pct} paused={false} />
                    </div>
                  </div>
                ))}
                <div className="flex items-center gap-4">
                  <span className="text-xs text-muted-foreground w-8 text-right">50%</span>
                  <div className="flex items-center gap-3">
                    <div className="space-y-0.5">
                      <ProgressMockup theme={theme} size="lg" progress={50} paused />
                      <span className="text-[10px] text-amber-400">paused</span>
                    </div>
                    <div className="space-y-0.5">
                      <ProgressMockup theme={theme} size="sm" progress={50} paused />
                      <span className="text-[10px] text-amber-400">paused</span>
                    </div>
                    <div className="space-y-0.5">
                      <ProgressMockup theme={theme} size="xs" progress={50} paused />
                      <span className="text-[10px] text-amber-400">paused</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          ))}
        </CardContent>
      </Card>

      {/* Section 5: Completion flash */}
      <Card>
        <CardHeader>
          <CardTitle>Completion Flash</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            The flash animation plays once on completion. Click to restart.
          </p>
          <div className="flex flex-wrap gap-6 items-start">
            <CompletedFlashDemo theme="movie" />
            <CompletedFlashDemo theme="tv" />
          </div>
        </CardContent>
      </Card>

      {/* Section 6: Monitor button styles */}
      <Card>
        <CardHeader>
          <CardTitle>Monitor Button Styles</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-sm text-muted-foreground">
            Monitor/Unmonitor button appearance across all sizes and themes.
          </p>
          <div className="grid grid-cols-2 gap-6">
            {(['movie', 'tv'] as const).map((theme) => (
              <div key={theme} className="space-y-3">
                <h3 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
                  {theme === 'movie' ? 'Movie' : 'TV'}
                </h3>
                <div className="space-y-2">
                  {(['lg', 'sm', 'xs'] as const).map((size) => (
                    <div key={size} className="flex items-center gap-4">
                      <span className="text-xs text-muted-foreground w-6">{size}</span>
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
    </div>
  )
}

// ---------------------------------------------------------------------------
// Live default row — actual component instances
// ---------------------------------------------------------------------------

function LiveDefaultRow({ theme, size }: { theme: MediaTheme; size: ControlSize }) {
  const [monitored, setMonitored] = useState(true)

  const commonProps = {
    theme,
    size,
    monitored,
    onMonitoredChange: setMonitored,
    qualityProfileId: 1,
    tmdbId: 550,
  }

  return (
    <div className="flex items-center gap-4">
      <Badge variant="outline" className="w-8 justify-center text-xs">{size}</Badge>
      {theme === 'movie' ? (
        <MediaSearchMonitorControls
          mediaType="movie"
          movieId={1}
          title="The Matrix"
          imdbId="tt0133093"
          year={1999}
          {...commonProps}
        />
      ) : (
        <MediaSearchMonitorControls
          mediaType="series"
          seriesId={1}
          title="Breaking Bad"
          tvdbId={81189}
          imdbId="tt0903747"
          {...commonProps}
        />
      )}
      <span className="text-xs text-muted-foreground">
        {monitored ? 'monitored' : 'unmonitored'}
      </span>
    </div>
  )
}

// ---------------------------------------------------------------------------
// State types
// ---------------------------------------------------------------------------

type StateName =
  | 'default-monitored'
  | 'default-unmonitored'
  | 'searching-auto'
  | 'searching-manual'
  | 'progress-35'
  | 'progress-72'
  | 'progress-paused'
  | 'completed'
  | 'error-notfound'
  | 'error-failed'

// ---------------------------------------------------------------------------
// Interactive state preview
// ---------------------------------------------------------------------------

type DemoState =
  | { type: 'default' }
  | { type: 'searching'; mode: 'manual' | 'auto' }
  | { type: 'progress'; percent: number }
  | { type: 'completed' }
  | { type: 'error'; message: string }

function InteractiveStatePreview() {
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
          <p className="text-sm text-muted-foreground">Click buttons to trigger state transitions</p>
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
  const [state, setState] = useState<DemoState>({ type: 'default' })
  const [monitored, setMonitored] = useState(true)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const cleanup = useCallback(() => {
    if (timerRef.current) { clearTimeout(timerRef.current); timerRef.current = null }
    if (intervalRef.current) { clearInterval(intervalRef.current); intervalRef.current = null }
  }, [])

  useEffect(() => cleanup, [cleanup])

  const runAutoSearch = useCallback(() => {
    cleanup()
    setState({ type: 'searching', mode: 'auto' })
    timerRef.current = setTimeout(() => {
      let pct = 0
      setState({ type: 'progress', percent: 0 })
      intervalRef.current = setInterval(() => {
        pct += 2
        if (pct >= 100) {
          if (intervalRef.current) clearInterval(intervalRef.current)
          setState({ type: 'completed' })
          timerRef.current = setTimeout(() => setState({ type: 'default' }), 2500)
        } else {
          setState({ type: 'progress', percent: pct })
        }
      }, 100)
    }, 2000)
  }, [cleanup])

  const runManualSearch = useCallback(() => {
    cleanup()
    setState({ type: 'searching', mode: 'manual' })
    timerRef.current = setTimeout(() => setState({ type: 'default' }), 2000)
  }, [cleanup])

  const isDefault = state.type === 'default'
  const gap = size === 'lg' ? 'gap-2' : size === 'sm' ? 'gap-1.5' : 'gap-1'

  const content = (() => {
    switch (state.type) {
      case 'searching':
        return <SearchingMockup theme={theme} size={size} mode={state.mode} fullWidth />
      case 'progress':
        return <ProgressMockup theme={theme} size={size} progress={state.percent} paused={false} fullWidth />
      case 'completed':
        return <CompletedMockup theme={theme} size={size} fullWidth />
      case 'error':
        return <ErrorMockup theme={theme} size={size} message={state.message} fullWidth />
      default:
        return null
    }
  })()

  return (
    <div className="flex items-center gap-4">
      <span className="text-xs text-muted-foreground font-medium w-6">{size}</span>
      <div className="flex-1 relative">
        <div className={cn('flex items-center', gap, !isDefault && 'invisible')}>
          <DefaultMockup theme={theme} size={size} monitored={monitored}
            onManualSearch={runManualSearch}
            onAutoSearch={runAutoSearch}
            onMonitoredChange={setMonitored}
          />
        </div>
        {!isDefault && (
          <div className="absolute inset-0 flex items-center">
            {content}
          </div>
        )}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// State mockup row for the grid
// ---------------------------------------------------------------------------

function StateRow({ label, state }: { label: string; state: StateName }) {
  return (
    <tr className="border-b last:border-0">
      <td className="py-3 px-3 font-medium text-xs">{label}</td>
      {(['movie', 'tv'] as const).map((theme) =>
        (['lg', 'sm', 'xs'] as const).map((size) => (
          <td key={`${theme}-${size}`} className="py-3 px-3">
            <div className="flex justify-center">
              <StateMockup state={state} theme={theme} size={size} />
            </div>
          </td>
        ))
      )}
    </tr>
  )
}

function StateMockup({ state, theme, size }: { state: StateName; theme: MediaTheme; size: ControlSize }) {
  const isDefault = state === 'default-monitored' || state === 'default-unmonitored'

  const fw = !isDefault
  const content = (() => {
    switch (state) {
      case 'default-monitored':
        return <DefaultMockup theme={theme} size={size} monitored />
      case 'default-unmonitored':
        return <DefaultMockup theme={theme} size={size} monitored={false} />
      case 'searching-auto':
        return <SearchingMockup theme={theme} size={size} mode="auto" fullWidth={fw} />
      case 'searching-manual':
        return <SearchingMockup theme={theme} size={size} mode="manual" fullWidth={fw} />
      case 'progress-35':
        return <ProgressMockup theme={theme} size={size} progress={35} paused={false} fullWidth={fw} />
      case 'progress-72':
        return <ProgressMockup theme={theme} size={size} progress={72} paused={false} fullWidth={fw} />
      case 'progress-paused':
        return <ProgressMockup theme={theme} size={size} progress={50} paused fullWidth={fw} />
      case 'completed':
        return <CompletedMockup theme={theme} size={size} fullWidth={fw} />
      case 'error-notfound':
        return <ErrorMockup theme={theme} size={size} message="Not Found" fullWidth={fw} />
      case 'error-failed':
        return <ErrorMockup theme={theme} size={size} message="Failed" fullWidth={fw} />
    }
  })()

  if (isDefault) return content

  // Non-default: show inside a container sized by invisible default buttons
  const gap = size === 'lg' ? 'gap-2' : size === 'sm' ? 'gap-1.5' : 'gap-1'
  return (
    <div className="relative">
      <div className={cn('flex items-center invisible', gap)}>
        <DefaultMockup theme={theme} size={size} monitored />
      </div>
      <div className="absolute inset-0 flex items-center">
        {content}
      </div>
    </div>
  )
}

// ---------------------------------------------------------------------------
// Visual mockups of each state
// ---------------------------------------------------------------------------

function DefaultMockup({ theme, size, monitored, onManualSearch, onAutoSearch, onMonitoredChange }: {
  theme: MediaTheme; size: ControlSize; monitored: boolean
  onManualSearch?: () => void; onAutoSearch?: () => void; onMonitoredChange?: (v: boolean) => void
}) {
  const interactive = !!onManualSearch
  const gap = size === 'lg' ? 'gap-2' : size === 'sm' ? 'gap-1.5' : 'gap-1'

  if (size === 'lg') {
    return (
      <div className={cn('flex items-center', gap)}>
        <Button variant="outline" size="default" disabled={!interactive} onClick={onManualSearch}>
          <UserSearch className="size-4 mr-2" />
          Search
        </Button>
        <Button variant="outline" size="default" disabled={!interactive} onClick={onAutoSearch}>
          <Zap className="size-4 mr-2" />
          Auto Search
        </Button>
        <Button variant="outline" size="default" disabled={!interactive} onClick={() => onMonitoredChange?.(!monitored)}>
          <span className="inline-grid [&>*]:col-start-1 [&>*]:row-start-1">
            <span className={cn('flex items-center', !monitored && 'invisible')}>
              <Eye className={cn('size-4 mr-2', theme === 'movie' ? 'text-movie-400' : 'text-tv-400')} />
              Monitored
            </span>
            <span className={cn('flex items-center', monitored && 'invisible')}>
              <EyeOff className="size-4 mr-2" />
              Unmonitored
            </span>
          </span>
        </Button>
      </div>
    )
  }

  if (size === 'sm') {
    return (
      <div className={cn('flex items-center', gap)}>
        <Button variant="outline" size="icon-sm" disabled={!interactive} onClick={onManualSearch}>
          <UserSearch className="size-4" />
        </Button>
        <Button variant="outline" size="icon-sm" disabled={!interactive} onClick={onAutoSearch}>
          <Zap className="size-4" />
        </Button>
        <Button variant="outline" size="icon-sm" disabled={!interactive} onClick={() => onMonitoredChange?.(!monitored)}>
          {monitored ? (
            <Eye className={cn('size-4', theme === 'movie' ? 'text-movie-400' : 'text-tv-400')} />
          ) : (
            <EyeOff className="size-4" />
          )}
        </Button>
      </div>
    )
  }

  // xs
  return (
    <div className={cn('flex items-center', gap)}>
      <Button variant="ghost" size="icon-sm" disabled={!interactive} onClick={onManualSearch}>
        <UserSearch className="size-3.5" />
      </Button>
      <Button variant="ghost" size="icon-sm" disabled={!interactive} onClick={onAutoSearch}>
        <Zap className="size-3.5" />
      </Button>
      <Button variant="ghost" size="icon-sm" disabled={!interactive} onClick={() => onMonitoredChange?.(!monitored)}>
        {monitored ? (
          <Eye className={cn('size-3.5', theme === 'movie' ? 'text-movie-400' : 'text-tv-400')} />
        ) : (
          <EyeOff className="size-3.5" />
        )}
      </Button>
    </div>
  )
}

function SearchingMockup({ theme, size, mode, fullWidth }: { theme: MediaTheme; size: ControlSize; mode: 'manual' | 'auto'; fullWidth?: boolean }) {
  if (mode === 'manual') {
    return (
      <Button variant="outline" disabled className={cn(fullWidth && 'w-full', size === 'xs' ? 'h-6 text-xs' : size === 'sm' ? 'h-8 text-xs' : '')}>
        Searching...
      </Button>
    )
  }

  // Auto
  if (size === 'xs') {
    const shimmerClass = theme === 'movie' ? 'shimmer-text-movie' : 'shimmer-text-tv'
    return (
      <Button variant="ghost" disabled className={cn('h-6 text-xs disabled:opacity-100', fullWidth && 'w-full')}>
        <span className={shimmerClass} data-text="Searching...">Searching...</span>
      </Button>
    )
  }

  const chasingClass = theme === 'movie' ? 'chasing-lights-movie' : 'chasing-lights-tv'

  if (size === 'lg') {
    return (
      <div className={cn(chasingClass, fullWidth && 'w-full')}>
        <div className="absolute inset-0 rounded-md bg-card z-[1]" />
        <Button variant="outline" disabled className="relative z-[2] w-full">
          Searching...
        </Button>
      </div>
    )
  }

  // sm
  return (
    <div className={cn(chasingClass, fullWidth && 'w-full')}>
      <div className="absolute inset-0 rounded-md bg-card z-[1]" />
      <Button variant="outline" disabled className={cn('relative z-[2] h-8 text-xs', fullWidth && 'w-full')}>
        Searching...
      </Button>
    </div>
  )
}

function ProgressMockup({ theme, size, progress, paused, fullWidth }: { theme: MediaTheme; size: ControlSize; progress: number; paused: boolean; fullWidth?: boolean }) {
  return (
    <div
      className={cn(
        'relative overflow-hidden rounded-md',
        paused && 'animation-paused',
        fullWidth
          ? cn('w-full', size === 'xs' ? 'h-6' : size === 'sm' ? 'h-8' : 'h-9')
          : (size === 'xs' ? 'h-6 w-20' : size === 'sm' ? 'h-8 w-24' : 'h-9 min-w-32'),
      )}
    >
      <div className="absolute inset-0 bg-muted/30" />
      <div
        className={cn(
          'absolute inset-y-0 left-0 transition-all duration-500 ease-out',
          theme === 'movie'
            ? 'bg-gradient-to-r from-movie-600/40 via-movie-500/50 to-movie-500/60'
            : 'bg-gradient-to-r from-tv-600/40 via-tv-500/50 to-tv-500/60',
        )}
        style={{ width: `${Math.max(progress, 2)}%` }}
      >
        {size !== 'xs' && (
          <div className="absolute inset-0 overflow-hidden">
            <div
              className={cn(
                'absolute inset-y-0 w-12 animate-[shimmer_1.5s_linear_infinite]',
                theme === 'movie'
                  ? 'bg-gradient-to-r from-transparent via-movie-400/25 to-transparent'
                  : 'bg-gradient-to-r from-transparent via-tv-400/25 to-transparent',
              )}
            />
          </div>
        )}
      </div>
      {size !== 'xs' && (
        <div
          className={cn(
            'absolute top-0 bottom-0 w-1 rounded-full blur-sm transition-all duration-500',
            theme === 'movie' ? 'bg-movie-400' : 'bg-tv-400',
          )}
          style={{ left: `calc(${Math.max(progress, 2)}% - 2px)` }}
        />
      )}
      {size !== 'xs' && (
        <div
          className={cn(
            'absolute inset-0 rounded-md ring-1 ring-inset',
            theme === 'movie'
              ? 'ring-movie-500/40 animate-[inset-glow-pulse-movie_2s_ease-in-out_infinite]'
              : 'ring-tv-500/40 animate-[inset-glow-pulse-tv_2s_ease-in-out_infinite]',
          )}
        />
      )}
      <div className="absolute inset-0 flex items-center justify-center gap-2 text-sm text-muted-foreground">
        <Download className={size === 'xs' ? 'size-3.5' : 'size-4'} />
        {size === 'lg' && 'Downloading'}
      </div>
    </div>
  )
}

function CompletedMockup({ theme, size, fullWidth }: { theme: MediaTheme; size: ControlSize; fullWidth?: boolean }) {
  const colorClass = theme === 'movie' ? 'text-movie-400' : 'text-tv-400'

  if (size === 'lg') {
    return (
      <Button variant="outline" disabled className={cn(fullWidth && 'w-full')}>
        <Check className={cn('size-4 mr-2', colorClass)} />
        Downloaded
      </Button>
    )
  }

  if (size === 'sm') {
    return (
      <Button variant="outline" size="icon-sm" disabled className={cn(fullWidth && 'w-full')}>
        <Check className={cn('size-4', colorClass)} />
      </Button>
    )
  }

  return (
    <div className={cn('p-1', fullWidth && 'w-full flex items-center justify-center')}>
      <Check className={cn('size-3.5', colorClass)} />
    </div>
  )
}

function ErrorMockup({ theme, size, message, fullWidth }: { theme: MediaTheme; size: ControlSize; message: string; fullWidth?: boolean }) {
  const colorClass = theme === 'movie' ? 'text-movie-400' : 'text-tv-400'

  if (size === 'lg') {
    return (
      <Button variant="outline" disabled className={cn(fullWidth && 'w-full')}>
        <AlertCircle className={cn('size-4 mr-2', colorClass)} />
        {message}
      </Button>
    )
  }

  if (size === 'sm') {
    return (
      <Tooltip>
        <TooltipTrigger
          render={<Button variant="outline" size="icon-sm" disabled className={cn(fullWidth && 'w-full')} />}
        >
          <AlertCircle className={cn('size-4', colorClass)} />
        </TooltipTrigger>
        <TooltipContent>{message}</TooltipContent>
      </Tooltip>
    )
  }

  return (
    <Tooltip>
      <TooltipTrigger>
        <div className={cn('p-1', fullWidth && 'w-full flex items-center justify-center')}>
          <AlertCircle className={cn('size-3.5', colorClass)} />
        </div>
      </TooltipTrigger>
      <TooltipContent>{message}</TooltipContent>
    </Tooltip>
  )
}

function MonitorMockup({ theme, size, monitored }: { theme: MediaTheme; size: ControlSize; monitored: boolean }) {
  const activeColor = theme === 'movie' ? 'text-movie-400' : 'text-tv-400'

  if (size === 'lg') {
    return (
      <Button variant="outline" disabled>
        <span className="inline-grid [&>*]:col-start-1 [&>*]:row-start-1">
          <span className={cn('flex items-center', !monitored && 'invisible')}>
            <Eye className={cn('size-4 mr-2', activeColor)} />
            Monitored
          </span>
          <span className={cn('flex items-center', monitored && 'invisible')}>
            <EyeOff className="size-4 mr-2" />
            Unmonitored
          </span>
        </span>
      </Button>
    )
  }

  if (size === 'sm') {
    return (
      <Button variant="outline" size="icon-sm" disabled>
        {monitored ? (
          <Eye className={cn('size-4', activeColor)} />
        ) : (
          <EyeOff className="size-4" />
        )}
      </Button>
    )
  }

  return (
    <Button variant="ghost" size="icon-sm" disabled>
      {monitored ? (
        <Eye className={cn('size-3.5', theme === 'movie' ? 'text-movie-400' : 'text-tv-400')} />
      ) : (
        <EyeOff className="size-3.5" />
      )}
    </Button>
  )
}

// ---------------------------------------------------------------------------
// Completion flash demo with replay
// ---------------------------------------------------------------------------

function CompletedFlashDemo({ theme }: { theme: MediaTheme }) {
  const [key, setKey] = useState(0)
  const flashClass = theme === 'movie'
    ? 'animate-[download-complete-flash-movie_800ms_ease-out]'
    : 'animate-[download-complete-flash-tv_800ms_ease-out]'
  const colorClass = theme === 'movie' ? 'text-movie-400' : 'text-tv-400'

  return (
    <div className="space-y-2">
      <Badge variant="outline">{theme === 'movie' ? 'Movie' : 'TV'}</Badge>
      <div className="flex items-center gap-3">
        <Button
          key={key}
          variant="outline"
          className={cn(flashClass, 'cursor-pointer')}
          onClick={() => setKey((k) => k + 1)}
        >
          <Check className={cn('size-4 mr-2', colorClass)} />
          Downloaded
        </Button>
        <Button
          key={`sm-${key}`}
          variant="outline"
          size="icon-sm"
          className={cn(flashClass, 'cursor-pointer')}
          onClick={() => setKey((k) => k + 1)}
        >
          <Check className={cn('size-4', colorClass)} />
        </Button>
        <span className="text-xs text-muted-foreground">click to replay</span>
      </div>
    </div>
  )
}
