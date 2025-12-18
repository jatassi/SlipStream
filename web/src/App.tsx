import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Film, Tv, Search, Download } from "lucide-react"

function App() {
  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b border-border">
        <div className="container mx-auto flex h-14 items-center px-4">
          <h1 className="text-xl font-semibold">SlipStream</h1>
          <nav className="ml-auto flex gap-2">
            <Button variant="ghost" size="sm">Movies</Button>
            <Button variant="ghost" size="sm">TV Shows</Button>
            <Button variant="ghost" size="sm">Settings</Button>
          </nav>
        </div>
      </header>

      {/* Main Content */}
      <main className="container mx-auto px-4 py-8">
        <div className="mb-8 text-center">
          <h2 className="text-3xl font-bold tracking-tight">Welcome to SlipStream</h2>
          <p className="mt-2 text-muted-foreground">
            Your unified media management solution
          </p>
        </div>

        {/* Feature Cards */}
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <Card>
            <CardHeader className="flex flex-row items-center gap-2">
              <Film className="size-5 text-primary" />
              <div>
                <CardTitle className="text-base">Movies</CardTitle>
                <CardDescription>Manage your movie library</CardDescription>
              </div>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                0 movies in library
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center gap-2">
              <Tv className="size-5 text-primary" />
              <div>
                <CardTitle className="text-base">TV Shows</CardTitle>
                <CardDescription>Manage your TV library</CardDescription>
              </div>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                0 series in library
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center gap-2">
              <Search className="size-5 text-primary" />
              <div>
                <CardTitle className="text-base">Indexers</CardTitle>
                <CardDescription>Search for releases</CardDescription>
              </div>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                0 indexers configured
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center gap-2">
              <Download className="size-5 text-primary" />
              <div>
                <CardTitle className="text-base">Downloads</CardTitle>
                <CardDescription>Monitor downloads</CardDescription>
              </div>
            </CardHeader>
            <CardContent>
              <p className="text-sm text-muted-foreground">
                0 active downloads
              </p>
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
