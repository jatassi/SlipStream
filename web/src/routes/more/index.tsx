import {
  Binoculars,
  Calendar,
  Cog,
  FileInput,
  FolderOpen,
  History,
  LogOut,
  RotateCcw,
  Server,
  Users,
  Workflow,
} from 'lucide-react'

import { Group, IconTile, Row } from '@/components/grouped-list'
import { DevModeControls } from '@/components/layout/dev-mode-controls'
import { LogoutDialog, RestartDialog } from '@/components/layout/sidebar-dialogs'
import { useHeader } from '@/components/layout/use-header'
import { useSessionActions } from '@/components/layout/use-session-actions'
import { Screen } from '@/components/screen/screen'
import { Skeleton } from '@/components/ui/skeleton'
import { useMissingCounts, useStatus } from '@/hooks'

function MissingTrailing() {
  const { data: counts, isLoading } = useMissingCounts()

  if (isLoading) {
    return <Skeleton className="h-4 w-12" />
  }

  return (
    <span className="nums flex items-center gap-1 text-footnote">
      <span className="text-movie-400">{counts?.movies ?? 0}</span>
      <span className="text-muted-foreground/50">|</span>
      <span className="text-tv-400">{counts?.episodes ?? 0}</span>
    </span>
  )
}

function DiscoverGroup() {
  return (
    <Group header="Discover">
      <Row leading={<IconTile className="bg-rose-600"><Calendar /></IconTile>} title="Calendar" href="/calendar" chevron />
      <Row leading={<IconTile className="bg-violet-600"><Users /></IconTile>} title="Requests" href="/requests-admin/queue" chevron />
      <Row
        leading={<IconTile className="bg-amber-600"><Binoculars /></IconTile>}
        title="Missing"
        href="/missing"
        trailing={<MissingTrailing />}
        chevron
      />
      <Row leading={<IconTile className="bg-sky-600"><FileInput /></IconTile>} title="Manual Import" href="/import" chevron />
      <Row leading={<IconTile className="bg-zinc-600"><History /></IconTile>} title="History" href="/history" chevron />
    </Group>
  )
}

function SettingsGroup() {
  return (
    <Group header="Settings">
      <Row leading={<IconTile className="bg-movie-600"><FolderOpen /></IconTile>} title="Media" href="/settings/media" chevron />
      <Row leading={<IconTile className="bg-tv-600"><Workflow /></IconTile>} title="Download Pipeline" href="/settings/download-pipeline" chevron />
      <Row leading={<IconTile className="bg-zinc-600"><Cog /></IconTile>} title="General" href="/settings/general" chevron />
      <Row leading={<IconTile className="bg-emerald-600"><Server /></IconTile>} title="System" href="/system/health" chevron />
    </Group>
  )
}

function DeveloperGroup() {
  const header = useHeader()
  if (!header.isDevBuild) {
    return null
  }
  return (
    <Group header="Developer Tools">
      <div className="px-4 py-2">
        <DevModeControls
          devModeEnabled={header.devModeEnabled}
          devModeSwitching={header.devModeSwitching}
          onToggle={header.handleDevModeToggle}
          globalLoading={header.globalLoading}
          onGlobalLoadingChange={header.setGlobalLoading}
        />
      </div>
    </Group>
  )
}

function SessionGroup({
  onLogout,
  onRestart,
}: {
  onLogout: () => void
  onRestart: () => void
}) {
  return (
    <Group>
      <Row leading={<LogOut className="size-5 text-amber-400" />} title="Log out" tone="warning" onClick={onLogout} />
      <Row leading={<RotateCcw className="size-5 text-destructive" />} title="Restart" tone="destructive" onClick={onRestart} />
    </Group>
  )
}

function VersionFooter() {
  const { data: status } = useStatus()
  if (!status) {
    return <Skeleton className="mx-auto mb-4 h-4 w-40" />
  }
  return (
    <p className="text-caption px-screen text-center text-muted-foreground">SlipStream {status.version}</p>
  )
}

export function MorePage() {
  const session = useSessionActions()

  return (
    <Screen title="More">
      <DiscoverGroup />
      <SettingsGroup />
      <DeveloperGroup />
      <SessionGroup
        onLogout={() => session.setShowLogoutDialog(true)}
        onRestart={() => session.setShowRestartDialog(true)}
      />
      <VersionFooter />
      <RestartDialog
        open={session.showRestartDialog}
        onOpenChange={session.setShowRestartDialog}
        onRestart={session.handleRestart}
        countdown={session.countdown}
        isPending={session.restartMutation.isPending}
      />
      <LogoutDialog
        open={session.showLogoutDialog}
        onOpenChange={session.setShowLogoutDialog}
        onLogout={session.handleLogout}
      />
    </Screen>
  )
}
