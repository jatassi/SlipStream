import type {
  RootsNodeVolumes,
  RootsRoot,
  TargetsProfile,
  WorksContentType,
  WorksPresetCounts,
} from "../../api/gen/model";
import { freeOf } from "../../roots/root-columns";
import { Select } from "../../ui/select";
import { Switch } from "../../ui/switch";
import { includeLabel } from "../monitoring/labels";
import { ProfilePicker } from "../profile-picker";
import type { AddChoices, MonitorControl } from "./choices";
import { PresetChoice } from "./preset-choice";

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

function MonitorField({
  type,
  monitor,
  counts,
  choices,
  choose,
}: {
  type: WorksContentType;
  monitor: MonitorControl;
  counts: WorksPresetCounts | null;
  choices: AddChoices;
  choose: Choose;
}) {
  if (monitor.kind === "choice") {
    return (
      <PresetChoice
        type={type}
        control={monitor}
        counts={counts}
        choices={choices}
        onChange={(preset) => {
          choose({ preset });
        }}
      />
    );
  }
  return (
    <Switch
      label="Monitor"
      checked={choices.preset !== "none"}
      onCheckedChange={(on) => {
        choose({ preset: on ? "all" : "none" });
      }}
    />
  );
}

function asideLabel(type: WorksContentType): string {
  if (type.aside === null) {
    throw new Error(`${type.id} has an add.asides setting and no aside`);
  }
  return includeLabel(type.aside);
}

function levelFoldersLabel(type: WorksContentType): string {
  const [outermost] = type.levels;
  if (outermost === undefined) {
    throw new Error(`${type.id} has an add.levelFolders setting and no levels`);
  }
  return `${outermost.one} folders`;
}

function LevelSwitches({
  type,
  choices,
  choose,
}: {
  type: WorksContentType;
  choices: AddChoices;
  choose: Choose;
}) {
  return (
    <>
      {choices.asides !== undefined && (
        <Switch
          label={asideLabel(type)}
          checked={choices.asides}
          onCheckedChange={(asides) => {
            choose({ asides });
          }}
        />
      )}
      {choices.levelFolders !== undefined && (
        <Switch
          label={levelFoldersLabel(type)}
          checked={choices.levelFolders}
          onCheckedChange={(levelFolders) => {
            choose({ levelFolders });
          }}
        />
      )}
    </>
  );
}

export function ChoiceFields({
  type,
  profiles,
  roots,
  volumes,
  monitor,
  counts,
  choices,
  moves,
  choose,
}: {
  type: WorksContentType;
  profiles: readonly TargetsProfile[];
  roots: readonly RootsRoot[];
  volumes: readonly RootsNodeVolumes[];
  monitor: MonitorControl;
  counts: WorksPresetCounts | null;
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
      <MonitorField
        type={type}
        monitor={monitor}
        counts={counts}
        choices={choices}
        choose={choose}
      />
      <LevelSwitches type={type} choices={choices} choose={choose} />
      <Remember type={type} remember={choices.remember} moves={moves} choose={choose} />
    </>
  );
}
