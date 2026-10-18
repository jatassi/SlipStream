import type { TargetsProfile } from "../api/gen/model";
import { Select } from "../ui/select";

export function ProfilePicker({
  profiles,
  value,
  onValueChange,
}: {
  profiles: readonly TargetsProfile[];
  value: string;
  onValueChange: (profileId: string) => void;
}) {
  return (
    <Select
      label="Profile"
      options={profiles.map((profile) => ({
        value: profile.id,
        label: profile.default ? `${profile.name} (default)` : profile.name,
      }))}
      value={value}
      onValueChange={onValueChange}
    />
  );
}
