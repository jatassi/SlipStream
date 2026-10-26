import { useState } from "react";

import type {
  TargetsProfile,
  TargetsQualityList,
  TargetsTarget,
  WorksContentType,
} from "../../api/gen/model";
import {
  useListProfilesSuspense,
  useListQualitiesSuspense,
  useListTargetsSuspense,
} from "../../api/gen/query/targets/targets";
import { useListContentTypesSuspense } from "../../api/gen/query/works/works";
import { SavedTargetEditor } from "../../targets/target-editor";
import { Dialog } from "../../ui/dialog";
import { StepFooter } from "./frame";
import type { StepBodyProps } from "./screens";
import { TargetLine } from "./targets-lines";

function TypeRow({
  type,
  profile,
  targets,
  qualities,
  onEdit,
}: {
  type: WorksContentType;
  profile: TargetsProfile;
  targets: readonly TargetsTarget[];
  qualities: TargetsQualityList;
  onEdit: (target: TargetsTarget) => void;
}) {
  const held = profile.targets.flatMap(
    (entry) => targets.find((target) => target.id === entry.targetId) ?? [],
  );
  return (
    <li aria-label={type.name} className="flex flex-col gap-2 px-4 py-3">
      <span className="font-semibold text-ink-1">{type.name}</span>
      {held.map((target) => (
        <TargetLine
          key={target.id}
          profile={profile}
          target={target}
          qualities={qualities}
          named={held.length > 1}
          onEdit={() => {
            onEdit(target);
          }}
        />
      ))}
    </li>
  );
}

// The Target stays in the dialog while it closes, so its content doesn't
// vanish mid-animation.
function TargetEditorDialog({
  editing,
  qualities,
  onClose,
}: {
  editing: { readonly target: TargetsTarget; readonly open: boolean } | undefined;
  qualities: TargetsQualityList;
  onClose: () => void;
}) {
  return (
    <Dialog
      title={`Edit ${editing?.target.name ?? ""}`}
      size={940}
      open={editing?.open === true}
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      {editing !== undefined && (
        <SavedTargetEditor
          key={editing.target.id}
          target={editing.target}
          qualities={qualities}
          onSaved={onClose}
        />
      )}
    </Dialog>
  );
}

export function TargetsStep({ nav }: StepBodyProps) {
  const { contentTypes } = useListContentTypesSuspense().data;
  const { profiles } = useListProfilesSuspense().data;
  const { targets } = useListTargetsSuspense().data;
  const qualities = useListQualitiesSuspense().data;
  const [editing, setEditing] = useState<{ target: TargetsTarget; open: boolean }>();
  const rows = contentTypes.flatMap((type) => {
    const profile = profiles.find((one) => one.contentType === type.id && one.default);
    return type.enabled && profile !== undefined ? [{ type, profile }] : [];
  });
  return (
    <>
      <ul
        aria-label="What SlipStream grabs"
        className="divide-y divide-hairline rounded-card bg-glass ring-1 ring-hairline ring-inset"
      >
        {rows.map(({ type, profile }) => (
          <TypeRow
            key={type.id}
            type={type}
            profile={profile}
            targets={targets}
            qualities={qualities}
            onEdit={(target) => {
              setEditing({ target, open: true });
            }}
          />
        ))}
      </ul>
      <StepFooter nav={nav} />
      <TargetEditorDialog
        editing={editing}
        qualities={qualities}
        onClose={() => {
          setEditing(editing && { ...editing, open: false });
        }}
      />
    </>
  );
}
