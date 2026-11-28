import type { TargetsQualityList, TargetsTarget } from "../api/gen/model";
import type { FormatName } from "./scores";
import { SavedConversionContext } from "./conversion-section";
import { refusedConversion } from "./conversion-words";
import { type TargetDraft, draftOf, problemsOf } from "./draft";
import { targetSections } from "./editor";
import { useTargetSave } from "./target-save";
import { type Save, SectionEditor } from "./section-editor";

export function TargetEditor({ initial, save }: { initial: TargetDraft; save: Save<TargetDraft> }) {
  return (
    <SectionEditor
      sections={targetSections}
      initial={initial}
      problemsOf={problemsOf}
      save={{
        label: save.label,
        onSave: async (draft) => {
          try {
            return await save.onSave(draft);
          } catch (error) {
            throw refusedConversion(error, draft);
          }
        },
      }}
    />
  );
}

/** Edits a saved Target, offering a remake while files made with older settings remain. */
export function SavedTargetEditor({
  target,
  qualities,
  formats,
  onSaved,
}: {
  target: TargetsTarget;
  qualities: TargetsQualityList;
  formats: readonly FormatName[];
  onSaved: () => Promise<void> | void;
}) {
  const { save, remake, dialogs } = useTargetSave(target, onSaved);
  return (
    <SavedConversionContext value={{ older: target.conversion?.older ?? 0, remake }}>
      <TargetEditor
        initial={draftOf(target, qualities, formats)}
        save={{ label: "Save", onSave: save }}
      />
      {dialogs}
    </SavedConversionContext>
  );
}
