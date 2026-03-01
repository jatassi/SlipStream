import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger } from '@/components/ui/select'

type ProfileSelectProps = {
  label: string
  value: number | null
  onChange: (id: number | null) => void
  qualityProfiles: { id: number; name: string }[] | undefined
}

export function ProfileSelect({ label, value, onChange, qualityProfiles }: ProfileSelectProps) {
  const profileLabel = value
    ? (qualityProfiles?.find((p) => p.id === value)?.name ?? 'Select profile')
    : 'Default (use global)'

  return (
    <div className="space-y-2">
      <Label>{label}</Label>
      <Select
        value={value?.toString() ?? ''}
        onValueChange={(v) => onChange(v ? Number.parseInt(v, 10) : null)}
      >
        <SelectTrigger>{profileLabel}</SelectTrigger>
        <SelectContent>
          <SelectItem value="">Default (use global)</SelectItem>
          {qualityProfiles?.map((profile) => (
            <SelectItem key={profile.id} value={profile.id.toString()}>
              {profile.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  )
}
