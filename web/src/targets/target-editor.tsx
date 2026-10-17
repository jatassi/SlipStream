import { type TargetDraft, problemsOf } from "./draft";
import { targetSections } from "./editor";
import { type Save, SectionEditor } from "./section-editor";

/** The Target editor, from targetSections. Without `save` it edits but offers no Save. */
export function TargetEditor({
  initial,
  save,
}: {
  initial: TargetDraft;
  save?: Save<TargetDraft>;
}) {
  return (
    <SectionEditor
      sections={targetSections}
      initial={initial}
      problemsOf={problemsOf}
      save={save}
    />
  );
}
