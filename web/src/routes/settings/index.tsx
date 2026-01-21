import { Link } from '@tanstack/react-router'
import {
  Sliders,
  FolderOpen,
  Rss,
  Download,
  Settings as SettingsIcon,
  ChevronRight,
  Search,
  FileInput,
  Layers,
  Bell,
  Users,
  ListTodo,
  Settings2,
  ExternalLink,
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
    title: 'Version Slots',
    description: 'Configure multi-version quality slots',
    href: '/settings/slots',
    icon: Layers,
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
    title: 'Notifications',
    description: 'Configure notification channels for events',
    href: '/settings/notifications',
    icon: Bell,
  },
  {
    title: 'Release Searching',
    description: 'Automatic search schedule and behavior',
    href: '/settings/autosearch',
    icon: Search,
  },
  {
    title: 'Import & Naming',
    description: 'File import validation and naming patterns',
    href: '/settings/import',
    icon: FileInput,
  },
  {
    title: 'General',
    description: 'Application settings and configuration',
    href: '/settings/general',
    icon: SettingsIcon,
  },
]

const requestsLinks = [
  {
    title: 'Request Queue',
    description: 'Review and approve content requests',
    href: '/settings/requests',
    icon: ListTodo,
  },
  {
    title: 'Portal Users',
    description: 'Manage portal users and invitations',
    href: '/settings/requests/users',
    icon: Users,
  },
  {
    title: 'Request Settings',
    description: 'Configure quotas, notifications, and rate limits',
    href: '/settings/requests/settings',
    icon: Settings2,
  },
]

export function SettingsPage() {
  return (
    <div className="space-y-8">
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

      <div>
        <h2 className="text-lg font-semibold mb-4">External Requests</h2>
        <div className="grid gap-4 md:grid-cols-2">
          {requestsLinks.map((link) => (
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
          <a href="/requests" target="_blank" rel="noopener noreferrer">
            <Card className="hover:border-primary transition-colors cursor-pointer">
              <CardHeader className="flex flex-row items-center gap-4">
                <div className="flex size-10 items-center justify-center rounded-lg bg-primary/10">
                  <ExternalLink className="size-5 text-primary" />
                </div>
                <div className="flex-1">
                  <CardTitle className="text-base">Open Portal</CardTitle>
                  <CardDescription>Open the request portal in a new tab</CardDescription>
                </div>
                <ExternalLink className="size-5 text-muted-foreground" />
              </CardHeader>
            </Card>
          </a>
        </div>
      </div>
    </div>
  )
}
