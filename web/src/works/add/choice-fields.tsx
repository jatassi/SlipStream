import type {
  RootsNodeVolumes,
  RootsRoot,
  TargetsProfile,
  WorksContentType,
} from "../../api/gen/model";
import { freeOf } from "../../roots/root-columns";
import { Select } from "../../ui/select";
import { Switch } from "../../ui/switch";
import { ProfilePicker } from "../profile-picker";
import type { AddChoices } from "./choices";

export const note = "text-meta text-ink-3 in-data-[density=touch]:text-body-touch";

type Choose = (next: Partial<AddChoices>) => void;

function RootPicker({
  roots,
  volumes,
  rootId,
  choose,
}: {
  roots: readonly RootsRoot[];
  volumes: readonly RootsNodeVolumes[];
  rootId: string;
  choose: Choose;
}) {
  return (
    <Select
      label="Library root"
      options={roots.map((root) => ({
        value: root.id,
        label: `${root.path} · ${freeOf(root, volumes)}`,
      }))}
      value={rootId}
      onValueChange={(next) => {
        choose({ rootId: next });
      }}
    />
  );
}

function Remember({
  type,
  remember,
  moves,
  choose,
}: {
  type: WorksContentType;
  remember: boolean;
  moves: boolean;
  choose: Choose;
}) {
  return (
    <div className="flex flex-col gap-1">
      <Switch
        label="Remember these choices"
        checked={remember}
        onCheckedChange={(next) => {
          choose({ remember: next });
        }}
      />
      {moves && <p className={note}>New {type.work.many} will start with these.</p>}
    </div>
  );
}

export function ChoiceFields({
  type,
  profiles,
  roots,
  volumes,
  choices,
  moves,
  choose,
}: {
  type: WorksContentType;
  profiles: readonly TargetsProfile[];
  roots: readonly RootsRoot[];
  volumes: readonly RootsNodeVolumes[];
  choices: AddChoices;
  moves: boolean;
  choose: Choose;
}) {
  return (
    <>
      <ProfilePicker
        profiles={profiles}
        value={choices.profileId}
        onValueChange={(profileId) => {
          choose({ profileId });
        }}
      />
      {roots.length >= 2 && choices.rootId !== undefined && (
        <RootPicker roots={roots} volumes={volumes} rootId={choices.rootId} choose={choose} />
      )}
      <Switch
        label="Monitor"
        checked={choices.monitor}
        onCheckedChange={(monitor) => {
          choose({ monitor });
        }}
      />
      <Remember type={type} remember={choices.remember} moves={moves} choose={choose} />
    </>
  );
}
