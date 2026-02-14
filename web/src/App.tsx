import { Download, Film, Search, Tv } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

function App() {
  return (
    <div className="bg-background min-h-screen">
      {/* Header */}
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

      {/* Main Content */}
      <main className="container mx-auto px-4 py-8">
        <div className="mb-8 text-center">
          <h2 className="text-3xl font-bold tracking-tight">Welcome to SlipStream</h2>
          <p className="text-muted-foreground mt-2">Your unified media management solution</p>
        </div>

        {/* Feature Cards */}
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <Card>
            <CardHeader className="flex flex-row items-center gap-2">
              <Film className="text-primary size-5" />
              <div>
                <CardTitle className="text-base">Movies</CardTitle>
                <CardDescription>Manage your movie library</CardDescription>
              </div>
            </CardHeader>
            <CardContent>
              <p className="text-muted-foreground text-sm">0 movies in library</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center gap-2">
              <Tv className="text-primary size-5" />
              <div>
                <CardTitle className="text-base">TV Shows</CardTitle>
                <CardDescription>Manage your TV library</CardDescription>
              </div>
            </CardHeader>
            <CardContent>
              <p className="text-muted-foreground text-sm">0 series in library</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center gap-2">
              <Search className="text-primary size-5" />
              <div>
                <CardTitle className="text-base">Indexers</CardTitle>
                <CardDescription>Search for releases</CardDescription>
              </div>
            </CardHeader>
            <CardContent>
              <p className="text-muted-foreground text-sm">0 indexers configured</p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center gap-2">
              <Download className="text-primary size-5" />
              <div>
                <CardTitle className="text-base">Downloads</CardTitle>
                <CardDescription>Monitor downloads</CardDescription>
              </div>
            </CardHeader>
            <CardContent>
              <p className="text-muted-foreground text-sm">0 active downloads</p>
            </CardContent>
          </Card>
        </div>

        {/* Quick Actions */}
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
