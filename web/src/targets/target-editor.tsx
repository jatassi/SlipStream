import { useNavigate } from "@tanstack/react-router";

import type { TargetsQualityList, TargetsTarget } from "../api/gen/model";
import { updateTarget } from "../api/gen/query/targets/targets";
import { toast } from "../ui/toast";
import { type TargetDraft, bodyOf, draftOf, problemsOf } from "./draft";
import { targetSections } from "./editor";
import { useImpactSheet } from "./impact-sheet";
import { type Save, SectionEditor } from "./section-editor";

export function TargetEditor({ initial, save }: { initial: TargetDraft; save: Save<TargetDraft> }) {
  return (
    <SectionEditor
      sections={targetSections}
      initial={initial}
      problemsOf={problemsOf}
      save={save}
    />
  );
}

export function SavedTargetEditor({
  target,
  qualities,
}: {
  target: TargetsTarget;
  qualities: TargetsQualityList;
}) {
  const navigate = useNavigate();
  const impact = useImpactSheet();
  return (
    <>
      <TargetEditor
        initial={draftOf(target, qualities)}
        save={{
          label: "Save",
          onSave: (draft) =>
            impact.save({
              title: `Save ${target.name}`,
              action: "Save",
              run: (changing) => updateTarget(target.id, bodyOf(draft), changing),
              done: async () => {
                toast(`Saved ${draft.name.trim()}.`);
                await navigate({ to: "/settings/$section", params: { section: "targets" } });
              },
            }),
        }}
      />
      {impact.sheet}
    </>
  );
}
