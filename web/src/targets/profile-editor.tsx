import { profileSections } from "./editor";
import { type ProfileDraft, profileProblemsOf } from "./profile-draft";
import { type Save, SectionEditor } from "./section-editor";

/** The Profile editor, from profileSections. Without `save` it edits but offers no Save. */
export function ProfileEditor({
  initial,
  save,
}: {
  initial: ProfileDraft;
  save?: Save<ProfileDraft>;
}) {
  return (
    <SectionEditor
      sections={profileSections}
      initial={initial}
      problemsOf={profileProblemsOf}
      save={save}
    />
  );
}
