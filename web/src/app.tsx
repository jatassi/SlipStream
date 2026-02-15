import type { LucideIcon } from 'lucide-react'
import { Download, Film, Search, Tv } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const FEATURE_CARDS = [
  { icon: Film, title: 'Movies', desc: 'Manage your movie library', stat: '0 movies in library' },
  { icon: Tv, title: 'TV Shows', desc: 'Manage your TV library', stat: '0 series in library' },
  { icon: Search, title: 'Indexers', desc: 'Search for releases', stat: '0 indexers configured' },
  { icon: Download, title: 'Downloads', desc: 'Monitor downloads', stat: '0 active downloads' },
] as const

function FeatureCard({
  icon: Icon,
  title,
  desc,
  stat,
}: {
  icon: LucideIcon
  title: string
  desc: string
  stat: string
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center gap-2">
        <Icon className="text-primary size-5" />
        <div>
          <CardTitle className="text-base">{title}</CardTitle>
          <CardDescription>{desc}</CardDescription>
        </div>
      </CardHeader>
      <CardContent>
        <p className="text-muted-foreground text-sm">{stat}</p>
      </CardContent>
    </Card>
  )
}

function App() {
  return (
    <div className="bg-background min-h-screen">
      <header className="border-border border-b">
        <div className="container mx-auto flex h-14 items-center px-4">
          <h1 className="text-xl font-semibold">SlipStream</h1>
          <nav className="ml-auto flex gap-2">
            <Button variant="ghost" size="sm">
              Movies
            </Button>
            <Button variant="ghost" size="sm">
              TV Shows
            </Button>
            <Button variant="ghost" size="sm">
              Settings
            </Button>
          </nav>
        </div>
      </header>
      <main className="container mx-auto px-4 py-8">
        <div className="mb-8 text-center">
          <h2 className="text-3xl font-bold tracking-tight">Welcome to SlipStream</h2>
          <p className="text-muted-foreground mt-2">Your unified media management solution</p>
        </div>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {FEATURE_CARDS.map((card) => (
            <FeatureCard key={card.title} {...card} />
          ))}
        </div>
        <div className="mt-8 flex justify-center gap-4">
          <Button>Add Movie</Button>
          <Button>Add TV Show</Button>
          <Button variant="outline">Configure Indexers</Button>
        </div>
      </main>
    </div>
  )
}

export default App
