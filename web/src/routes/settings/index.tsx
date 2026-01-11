import { Link } from '@tanstack/react-router'
import {
  Sliders,
  FolderOpen,
  Rss,
  Download,
  Settings as SettingsIcon,
  ChevronRight,
  Search,
} from 'lucide-react'
import { PageHeader } from '@/components/layout/PageHeader'
import { Card, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

const settingsLinks = [
  {
    title: 'Quality Profiles',
    description: 'Configure quality profiles for movies and series',
    href: '/settings/profiles',
    icon: Sliders,
  },
  {
    title: 'Root Folders',
    description: 'Manage media library storage locations',
    href: '/settings/rootfolders',
    icon: FolderOpen,
  },
  {
    title: 'Indexers',
    description: 'Configure search providers (Torznab/Newznab)',
    href: '/settings/indexers',
    icon: Rss,
  },
  {
    title: 'Download Clients',
    description: 'Set up torrent and usenet clients',
    href: '/settings/downloadclients',
    icon: Download,
  },
  {
    title: 'Release Searching',
    description: 'Automatic search schedule and behavior',
    href: '/settings/autosearch',
    icon: Search,
  },
  {
    title: 'General',
    description: 'Application settings and configuration',
    href: '/settings/general',
    icon: SettingsIcon,
  },
]

export function SettingsPage() {
  return (
    <div>
      <PageHeader
        title="Settings"
        description="Configure SlipStream"
      />

      <div className="grid gap-4 md:grid-cols-2">
        {settingsLinks.map((link) => (
          <Link key={link.href} to={link.href}>
            <Card className="hover:border-primary transition-colors cursor-pointer">
              <CardHeader className="flex flex-row items-center gap-4">
                <div className="flex size-10 items-center justify-center rounded-lg bg-primary/10">
                  <link.icon className="size-5 text-primary" />
                </div>
                <div className="flex-1">
                  <CardTitle className="text-base">{link.title}</CardTitle>
                  <CardDescription>{link.description}</CardDescription>
                </div>
                <ChevronRight className="size-5 text-muted-foreground" />
              </CardHeader>
            </Card>
          </Link>
        ))}
      </div>
    </div>
  )
}
