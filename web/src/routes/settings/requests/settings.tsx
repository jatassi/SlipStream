import { Loader2, Save } from 'lucide-react'

import { ErrorState } from '@/components/data/error-state'
import { LoadingState } from '@/components/data/loading-state'
import { PageHeader } from '@/components/layout/page-header'
import { NotificationDialog } from '@/components/notifications/notification-dialog'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import type { RequestSettings, RootFolder } from '@/types'

import { RequestsNav } from './requests-nav'
import { NotificationsCard } from './settings-notification-cards'
import { useRequestSettingsPage } from './use-settings-page'

export function RequestSettingsPage() {
  const page = useRequestSettingsPage()

  if (page.isLoading) {
    return (
      <div>
        <PageHeader title="Request Settings" />
        <LoadingState variant="card" />
      </div>
    )
  }

  if (page.isError) {
    return (
      <div>
        <PageHeader title="Request Settings" />
        <ErrorState onRetry={page.refetch} />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <SettingsPageHeader
        onSave={page.handleSave}
        hasChanges={page.hasChanges}
        isSaving={page.updateMutation.isPending}
      />
      <RequestsNav />
      <SettingsBody page={page} />
      <NotificationDialog
        open={page.showNotificationDialog}
        onOpenChange={page.setShowNotificationDialog}
        notification={page.editingNotification}
      />
    </div>
  )
}

function SettingsPageHeader(props: {
  onSave: () => void
  hasChanges: boolean
  isSaving: boolean
}) {
  return (
    <PageHeader
      title="External Requests"
      description="Manage portal users and content requests"
      breadcrumbs={[
        { label: 'Settings', href: '/settings/media' },
        { label: 'External Requests' },
      ]}
      actions={
        <Button onClick={props.onSave} disabled={!props.hasChanges || props.isSaving}>
          {props.isSaving ? (
            <Loader2 className="mr-2 size-4 animate-spin" />
          ) : (
            <Save className="mr-2 size-4" />
          )}
          Save Changes
        </Button>
      }
    />
  )
}

type PageState = ReturnType<typeof useRequestSettingsPage>

function SettingsBody({ page }: { page: PageState }) {
  return (
    <div className="space-y-6">
      <PortalAccessCard
        enabled={page.formData.enabled ?? true}
        onChange={(checked) => page.handleChange('enabled', checked)}
      />
      {page.portalEnabled ? (
        <>
          <QuotasCard formData={page.formData} onChange={page.handleChange} />
          <ContentSettingsCard
            formData={page.formData}
            rootFolders={page.rootFolders}
            onChange={page.handleChange}
          />
          <NotificationsCard
            notifications={page.notifications}
            formData={page.formData}
            onChange={page.handleChange}
            getTypeName={page.getTypeName}
            onAdd={page.handleOpenAddNotification}
            onEdit={page.handleOpenEditNotification}
            onTest={page.handleTestNotification}
            onDelete={page.handleDeleteNotification}
            onToggleEnabled={page.handleToggleNotificationEnabled}
            isTestPending={page.testNotificationMutation.isPending}
          />
          <RateLimitCard formData={page.formData} onChange={page.handleChange} />
        </>
      ) : null}
    </div>
  )
}

function PortalAccessCard(props: { enabled: boolean; onChange: (checked: boolean) => void }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Portal Access</CardTitle>
        <CardDescription>
          Enable or disable the external requests portal for all users.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex items-center justify-between">
          <div className="space-y-0.5">
            <Label>Enable External Requests Portal</Label>
            <p className="text-muted-foreground text-sm">
              When disabled, portal users cannot access the request system. Existing users and data
              are preserved.
            </p>
          </div>
          <Switch checked={props.enabled} onCheckedChange={props.onChange} />
        </div>
      </CardContent>
    </Card>
  )
}

type FormChangeHandler = <K extends keyof RequestSettings>(
  key: K,
  value: RequestSettings[K],
) => void

function QuotasCard(props: { formData: Partial<RequestSettings>; onChange: FormChangeHandler }) {
  const { formData, onChange } = props
  return (
    <Card>
      <CardHeader>
        <CardTitle>Default Quotas</CardTitle>
        <CardDescription>
          Set the default weekly quota limits for new users. Users can have individual overrides.
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-4 sm:grid-cols-3">
          <QuotaInput
            id="movieQuota"
            label="Movies per Week"
            value={formData.defaultMovieQuota}
            onChange={(v) => onChange('defaultMovieQuota', v)}
            placeholder="e.g., 5"
          />
          <QuotaInput
            id="seasonQuota"
            label="Seasons per Week"
            value={formData.defaultSeasonQuota}
            onChange={(v) => onChange('defaultSeasonQuota', v)}
            placeholder="e.g., 3"
          />
          <QuotaInput
            id="episodeQuota"
            label="Episodes per Week"
            value={formData.defaultEpisodeQuota}
            onChange={(v) => onChange('defaultEpisodeQuota', v)}
            placeholder="e.g., 10"
          />
        </div>
      </CardContent>
    </Card>
  )
}

function QuotaInput(props: {
  id: string
  label: string
  value: number | undefined
  onChange: (value: number) => void
  placeholder: string
}) {
  return (
    <div className="space-y-2">
      <Label htmlFor={props.id}>{props.label}</Label>
      <Input
        id={props.id}
        type="number"
        min="0"
        value={props.value ?? ''}
        onChange={(e) => props.onChange(Number.parseInt(e.target.value, 10) || 0)}
        placeholder={props.placeholder}
      />
      <p className="text-muted-foreground text-xs">Set to 0 for unlimited</p>
    </div>
  )
}

function ContentSettingsCard(props: {
  formData: Partial<RequestSettings>
  rootFolders: RootFolder[] | undefined
  onChange: FormChangeHandler
}) {
  const { formData, rootFolders, onChange } = props
  return (
    <Card>
      <CardHeader>
        <CardTitle>Content Settings</CardTitle>
        <CardDescription>Configure default settings for requested content.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label>Default Root Folder</Label>
          <Select
            value={formData.defaultRootFolderId?.toString() ?? ''}
            onValueChange={(value) =>
              onChange('defaultRootFolderId', value ? Number.parseInt(value, 10) : null)
            }
          >
            <SelectTrigger className="w-full max-w-md">
              {formData.defaultRootFolderId
                ? rootFolders?.find((f) => f.id === formData.defaultRootFolderId)?.path ??
                  'Selected folder'
                : 'Select default root folder'}
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="">No default (use first available)</SelectItem>
              {rootFolders?.map((folder) => (
                <SelectItem key={folder.id} value={folder.id.toString()}>
                  {folder.path}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <p className="text-muted-foreground text-xs">
            The root folder where requested content will be downloaded by default.
          </p>
        </div>
      </CardContent>
    </Card>
  )
}

function RateLimitCard(props: { formData: Partial<RequestSettings>; onChange: FormChangeHandler }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Rate Limiting</CardTitle>
        <CardDescription>Control search rate limits to prevent abuse.</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="rateLimit">Search Rate Limit</Label>
          <div className="flex max-w-md items-center gap-2">
            <Input
              id="rateLimit"
              type="number"
              min="1"
              max="100"
              value={props.formData.searchRateLimit ?? ''}
              onChange={(e) =>
                props.onChange('searchRateLimit', Number.parseInt(e.target.value, 10) || 10)
              }
            />
            <span className="text-muted-foreground text-sm whitespace-nowrap">
              requests per minute
            </span>
          </div>
          <p className="text-muted-foreground text-xs">
            Maximum number of search requests a user can make per minute. Applies globally to all
            portal users.
          </p>
        </div>
      </CardContent>
    </Card>
  )
}
