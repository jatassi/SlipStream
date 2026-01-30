import { PageHeader } from '@/components/layout/PageHeader'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress as ProgressPrimitive } from '@base-ui/react/progress'

const MOVIE_SHADES = [
  { shade: '50', bg: 'bg-movie-50' },
  { shade: '100', bg: 'bg-movie-100' },
  { shade: '200', bg: 'bg-movie-200' },
  { shade: '300', bg: 'bg-movie-300' },
  { shade: '400', bg: 'bg-movie-400' },
  { shade: '500', bg: 'bg-movie-500' },
  { shade: '600', bg: 'bg-movie-600' },
  { shade: '700', bg: 'bg-movie-700' },
  { shade: '800', bg: 'bg-movie-800' },
  { shade: '900', bg: 'bg-movie-900' },
  { shade: '950', bg: 'bg-movie-950' },
] as const

const TV_SHADES = [
  { shade: '50', bg: 'bg-tv-50' },
  { shade: '100', bg: 'bg-tv-100' },
  { shade: '200', bg: 'bg-tv-200' },
  { shade: '300', bg: 'bg-tv-300' },
  { shade: '400', bg: 'bg-tv-400' },
  { shade: '500', bg: 'bg-tv-500' },
  { shade: '600', bg: 'bg-tv-600' },
  { shade: '700', bg: 'bg-tv-700' },
  { shade: '800', bg: 'bg-tv-800' },
  { shade: '900', bg: 'bg-tv-900' },
  { shade: '950', bg: 'bg-tv-950' },
] as const

function ColorSwatch({ bg, shade, label }: { bg: string; shade: string; label: string }) {
  return (
    <div className="flex flex-col items-center gap-1">
      <div
        className={`w-16 h-16 rounded-md ${bg} ring-1 ring-white/10`}
        title={`${label}-${shade}`}
      />
      <span className="text-xs text-muted-foreground">{shade}</span>
    </div>
  )
}

function PaletteRow({ shades, label }: { shades: typeof MOVIE_SHADES; label: string }) {
  return (
    <div className="space-y-3">
      <h3 className="text-sm font-medium text-foreground">{label}</h3>
      <div className="flex flex-wrap gap-3">
        {shades.map(({ shade, bg }) => (
          <ColorSwatch key={shade} bg={bg} shade={shade} label={label} />
        ))}
      </div>
    </div>
  )
}

function GradientSwatch({ className, label }: { className: string; label: string }) {
  return (
    <div className="flex flex-col items-center gap-2">
      <div className={`w-48 h-16 rounded-md ${className} ring-1 ring-white/10`} />
      <span className="text-xs text-muted-foreground">{label}</span>
    </div>
  )
}

function ContrastTest() {
  return (
    <div className="space-y-4">
      <h3 className="text-sm font-medium text-foreground">Contrast on Dark Backgrounds</h3>
      <div className="grid grid-cols-2 gap-4">
        <div className="p-4 rounded-md bg-background space-y-2">
          <p className="text-xs text-muted-foreground mb-2">bg-background</p>
          <p className="text-movie-400">Movie 400</p>
          <p className="text-movie-500">Movie 500</p>
          <p className="text-tv-400">TV 400</p>
          <p className="text-tv-500">TV 500</p>
        </div>
        <div className="p-4 rounded-md bg-card space-y-2">
          <p className="text-xs text-muted-foreground mb-2">bg-card</p>
          <p className="text-movie-400">Movie 400</p>
          <p className="text-movie-500">Movie 500</p>
          <p className="text-tv-400">TV 400</p>
          <p className="text-tv-500">TV 500</p>
        </div>
        <div className="p-4 rounded-md bg-muted space-y-2">
          <p className="text-xs text-muted-foreground mb-2">bg-muted</p>
          <p className="text-movie-400">Movie 400</p>
          <p className="text-movie-500">Movie 500</p>
          <p className="text-tv-400">TV 400</p>
          <p className="text-tv-500">TV 500</p>
        </div>
        <div className="p-4 rounded-md bg-accent space-y-2">
          <p className="text-xs text-muted-foreground mb-2">bg-accent</p>
          <p className="text-movie-400">Movie 400</p>
          <p className="text-movie-500">Movie 500</p>
          <p className="text-tv-400">TV 400</p>
          <p className="text-tv-500">TV 500</p>
        </div>
      </div>
    </div>
  )
}

function ThemedProgress({
  value,
  theme,
  glow
}: {
  value: number
  theme: 'movie' | 'tv' | 'media'
  glow?: boolean
}) {
  const trackClass = theme === 'movie'
    ? 'bg-movie-950'
    : theme === 'tv'
      ? 'bg-tv-950'
      : 'bg-muted'

  const indicatorClass = theme === 'movie'
    ? `bg-movie-500 ${glow ? 'glow-movie-sm' : ''}`
    : theme === 'tv'
      ? `bg-tv-500 ${glow ? 'glow-tv-sm' : ''}`
      : `bg-media-gradient ${glow ? 'glow-media-sm' : ''}`

  return (
    <ProgressPrimitive.Root value={value} className="w-full">
      <ProgressPrimitive.Track className={`h-2 rounded-full relative overflow-hidden ${trackClass}`}>
        <ProgressPrimitive.Indicator className={`h-full rounded-full transition-all ${indicatorClass}`} />
      </ProgressPrimitive.Track>
    </ProgressPrimitive.Root>
  )
}

export function ColorPreviewPage() {
  return (
    <div className="space-y-8">
      <PageHeader
        title="Color Palette Preview"
        description="Preview the movie (orange) and TV (blue) color palettes"
      />

      {/* Palettes */}
      <section className="space-y-6">
        <h2 className="text-lg font-semibold">Color Palettes</h2>
        <PaletteRow shades={MOVIE_SHADES} label="Movie (Orange)" />
        <PaletteRow shades={TV_SHADES} label="TV (Blue)" />
      </section>

      {/* Gradients */}
      <section className="space-y-4">
        <h2 className="text-lg font-semibold">Gradients (Movie → TV)</h2>
        <div className="flex flex-wrap gap-6">
          <GradientSwatch className="bg-media-gradient" label="bg-media-gradient" />
          <GradientSwatch className="bg-media-gradient-vibrant" label="bg-media-gradient-vibrant" />
          <GradientSwatch className="bg-media-gradient-muted" label="bg-media-gradient-muted" />
        </div>
      </section>

      {/* Text Examples */}
      <section className="space-y-4">
        <h2 className="text-lg font-semibold">Text Colors</h2>
        <div className="grid gap-6 md:grid-cols-3">
          <div className="space-y-2">
            <h4 className="text-sm text-muted-foreground">Movie</h4>
            <p className="text-movie-400 text-lg">movie-400: Light accent</p>
            <p className="text-movie-500 text-lg">movie-500: Standard</p>
            <p className="text-movie-600 text-lg">movie-600: Darker</p>
          </div>
          <div className="space-y-2">
            <h4 className="text-sm text-muted-foreground">TV</h4>
            <p className="text-tv-400 text-lg">tv-400: Light accent</p>
            <p className="text-tv-500 text-lg">tv-500: Standard</p>
            <p className="text-tv-600 text-lg">tv-600: Darker</p>
          </div>
          <div className="space-y-2">
            <h4 className="text-sm text-muted-foreground">Gradient</h4>
            <p className="text-media-gradient text-xl font-semibold">
              Gradient Text Example
            </p>
          </div>
        </div>
      </section>

      {/* Interactive Elements - Buttons */}
      <section className="space-y-4">
        <h2 className="text-lg font-semibold">Buttons</h2>
        <div className="space-y-6">
          <div className="space-y-3">
            <h4 className="text-sm text-muted-foreground">Movie Buttons</h4>
            <div className="flex flex-wrap gap-3">
              <Button className="bg-movie-500 text-white hover:bg-movie-400 border-movie-500">
                Primary
              </Button>
              <Button variant="outline" className="border-movie-500/50 text-movie-400 hover:bg-movie-500/10 hover:text-movie-300">
                Outline
              </Button>
              <Button variant="secondary" className="bg-movie-500/20 text-movie-400 hover:bg-movie-500/30">
                Secondary
              </Button>
              <Button variant="ghost" className="text-movie-400 hover:bg-movie-500/10 hover:text-movie-300">
                Ghost
              </Button>
            </div>
          </div>
          <div className="space-y-3">
            <h4 className="text-sm text-muted-foreground">TV Buttons</h4>
            <div className="flex flex-wrap gap-3">
              <Button className="bg-tv-500 text-white hover:bg-tv-400 border-tv-500">
                Primary
              </Button>
              <Button variant="outline" className="border-tv-500/50 text-tv-400 hover:bg-tv-500/10 hover:text-tv-300">
                Outline
              </Button>
              <Button variant="secondary" className="bg-tv-500/20 text-tv-400 hover:bg-tv-500/30">
                Secondary
              </Button>
              <Button variant="ghost" className="text-tv-400 hover:bg-tv-500/10 hover:text-tv-300">
                Ghost
              </Button>
            </div>
          </div>
          <div className="space-y-3">
            <h4 className="text-sm text-muted-foreground">Gradient Buttons</h4>
            <div className="flex flex-wrap gap-3">
              <Button className="bg-media-gradient text-white border-0 hover:opacity-90">
                Gradient
              </Button>
              <Button variant="outline" className="border-movie-500/50 text-media-gradient hover:bg-muted">
                Gradient Text
              </Button>
            </div>
          </div>
        </div>
      </section>

      {/* Badges */}
      <section className="space-y-4">
        <h2 className="text-lg font-semibold">Badges</h2>
        <div className="flex flex-wrap gap-8">
          <div className="space-y-3">
            <h4 className="text-sm text-muted-foreground">Movie</h4>
            <div className="flex flex-wrap gap-2">
              <Badge className="bg-movie-500 text-white">Solid</Badge>
              <Badge variant="secondary" className="bg-movie-500/20 text-movie-400">Secondary</Badge>
              <Badge variant="outline" className="border-movie-500/50 text-movie-400">Outline</Badge>
            </div>
          </div>
          <div className="space-y-3">
            <h4 className="text-sm text-muted-foreground">TV</h4>
            <div className="flex flex-wrap gap-2">
              <Badge className="bg-tv-500 text-white">Solid</Badge>
              <Badge variant="secondary" className="bg-tv-500/20 text-tv-400">Secondary</Badge>
              <Badge variant="outline" className="border-tv-500/50 text-tv-400">Outline</Badge>
            </div>
          </div>
          <div className="space-y-3">
            <h4 className="text-sm text-muted-foreground">Gradient</h4>
            <div className="flex flex-wrap gap-2">
              <Badge className="bg-media-gradient text-white">Gradient</Badge>
              <Badge variant="outline" className="text-media-gradient border-movie-500/30">Gradient Text</Badge>
            </div>
          </div>
        </div>
      </section>

      {/* Contrast Test */}
      <section className="space-y-4">
        <h2 className="text-lg font-semibold">Contrast Testing</h2>
        <ContrastTest />
      </section>

      {/* Border Examples */}
      <section className="space-y-4">
        <h2 className="text-lg font-semibold">Borders & Rings</h2>
        <div className="flex gap-4">
          <div className="w-24 h-24 rounded-md border-2 border-movie-500 flex items-center justify-center text-sm text-muted-foreground">
            Movie
          </div>
          <div className="w-24 h-24 rounded-md border-2 border-tv-500 flex items-center justify-center text-sm text-muted-foreground">
            TV
          </div>
          <div className="w-24 h-24 rounded-md ring-2 ring-movie-500 flex items-center justify-center text-sm text-muted-foreground">
            Ring
          </div>
          <div className="w-24 h-24 rounded-md ring-2 ring-tv-500 flex items-center justify-center text-sm text-muted-foreground">
            Ring
          </div>
        </div>
      </section>

      {/* Glow Effects */}
      <section className="space-y-6">
        <h2 className="text-lg font-semibold">Glow Effects</h2>

        {/* Movie Glows */}
        <div className="space-y-3">
          <h4 className="text-sm text-muted-foreground">Movie Glows</h4>
          <div className="flex flex-wrap gap-6">
            <div className="w-24 h-24 rounded-md bg-movie-950 glow-movie-sm flex items-center justify-center text-sm text-movie-400">
              sm
            </div>
            <div className="w-24 h-24 rounded-md bg-movie-950 glow-movie flex items-center justify-center text-sm text-movie-400">
              default
            </div>
            <div className="w-24 h-24 rounded-md bg-movie-950 glow-movie-lg flex items-center justify-center text-sm text-movie-400">
              lg
            </div>
            <div className="w-24 h-24 rounded-md bg-movie-950 glow-movie-border flex items-center justify-center text-sm text-movie-400">
              border
            </div>
            <div className="w-24 h-24 rounded-md bg-movie-950 glow-movie-ring flex items-center justify-center text-sm text-movie-400">
              ring
            </div>
            <div className="w-24 h-24 rounded-md bg-movie-950 glow-movie-pulse flex items-center justify-center text-sm text-movie-400">
              pulse
            </div>
          </div>
        </div>

        {/* TV Glows */}
        <div className="space-y-3">
          <h4 className="text-sm text-muted-foreground">TV Glows</h4>
          <div className="flex flex-wrap gap-6">
            <div className="w-24 h-24 rounded-md bg-tv-950 glow-tv-sm flex items-center justify-center text-sm text-tv-400">
              sm
            </div>
            <div className="w-24 h-24 rounded-md bg-tv-950 glow-tv flex items-center justify-center text-sm text-tv-400">
              default
            </div>
            <div className="w-24 h-24 rounded-md bg-tv-950 glow-tv-lg flex items-center justify-center text-sm text-tv-400">
              lg
            </div>
            <div className="w-24 h-24 rounded-md bg-tv-950 glow-tv-border flex items-center justify-center text-sm text-tv-400">
              border
            </div>
            <div className="w-24 h-24 rounded-md bg-tv-950 glow-tv-ring flex items-center justify-center text-sm text-tv-400">
              ring
            </div>
            <div className="w-24 h-24 rounded-md bg-tv-950 glow-tv-pulse flex items-center justify-center text-sm text-tv-400">
              pulse
            </div>
          </div>
        </div>

        {/* Media Gradient Glows */}
        <div className="space-y-3">
          <h4 className="text-sm text-muted-foreground">Media Gradient Glows (both colors)</h4>
          <div className="flex flex-wrap gap-6">
            <div className="w-32 h-24 rounded-md bg-card glow-media-sm flex items-center justify-center text-sm text-muted-foreground">
              sm
            </div>
            <div className="w-32 h-24 rounded-md bg-card glow-media flex items-center justify-center text-sm text-muted-foreground">
              default
            </div>
            <div className="w-32 h-24 rounded-md bg-card glow-media-lg flex items-center justify-center text-sm text-muted-foreground">
              lg
            </div>
            <div className="w-32 h-24 rounded-md bg-card glow-media-pulse flex items-center justify-center text-sm text-muted-foreground">
              pulse
            </div>
          </div>
        </div>
      </section>

      {/* Progress Bars with Glow */}
      <section className="space-y-4">
        <h2 className="text-lg font-semibold">Progress Bars with Glow</h2>
        <div className="space-y-6 max-w-md">
          <div className="space-y-3">
            <h4 className="text-sm text-muted-foreground">Movie Progress</h4>
            <ThemedProgress value={75} theme="movie" />
            <ThemedProgress value={60} theme="movie" glow />
          </div>
          <div className="space-y-3">
            <h4 className="text-sm text-muted-foreground">TV Progress</h4>
            <ThemedProgress value={50} theme="tv" />
            <ThemedProgress value={80} theme="tv" glow />
          </div>
          <div className="space-y-3">
            <h4 className="text-sm text-muted-foreground">Media Gradient Progress</h4>
            <ThemedProgress value={65} theme="media" />
            <ThemedProgress value={45} theme="media" glow />
          </div>
        </div>
      </section>

      {/* Glowing Cards */}
      <section className="space-y-4">
        <h2 className="text-lg font-semibold">Glowing Cards</h2>
        <div className="flex flex-wrap gap-6">
          <Card className="w-56 border-movie-500/30 glow-movie-sm hover:glow-movie transition-shadow">
            <CardHeader>
              <CardTitle className="text-movie-400">Movie Card</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">Hover for stronger glow effect</p>
            </CardContent>
          </Card>
          <Card className="w-56 border-tv-500/30 glow-tv-sm hover:glow-tv transition-shadow">
            <CardHeader>
              <CardTitle className="text-tv-400">TV Card</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">Hover for stronger glow effect</p>
            </CardContent>
          </Card>
          <Card className="w-56 glow-media-sm hover:glow-media transition-shadow">
            <CardHeader>
              <CardTitle className="text-media-gradient">Media Card</CardTitle>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">Dual color glow effect</p>
            </CardContent>
          </Card>
        </div>
      </section>

      {/* Usage Examples */}
      <section className="space-y-4">
        <h2 className="text-lg font-semibold">Usage Reference</h2>
        <Card>
          <CardContent className="font-mono text-sm space-y-2 pt-4">
            <p className="text-muted-foreground">{`// Background colors`}</p>
            <p>bg-movie-500, bg-tv-500, bg-media-gradient</p>
            <p className="text-muted-foreground mt-3">{`// Text colors`}</p>
            <p>text-movie-400, text-tv-400, text-media-gradient</p>
            <p className="text-muted-foreground mt-3">{`// Borders & rings`}</p>
            <p>border-movie-500, border-tv-500, ring-movie-500, ring-tv-500</p>
            <p className="text-muted-foreground mt-3">{`// With opacity`}</p>
            <p>bg-movie-500/20, text-tv-400/80</p>
            <p className="text-muted-foreground mt-3">{`// Glow effects`}</p>
            <p>glow-movie-sm, glow-movie, glow-movie-lg, glow-movie-border, glow-movie-ring</p>
            <p>glow-tv-sm, glow-tv, glow-tv-lg, glow-tv-border, glow-tv-ring</p>
            <p>glow-media-sm, glow-media, glow-media-lg</p>
            <p className="text-muted-foreground mt-3">{`// Animated glows`}</p>
            <p>glow-movie-pulse, glow-tv-pulse, glow-media-pulse</p>
          </CardContent>
        </Card>
      </section>
    </div>
  )
}
