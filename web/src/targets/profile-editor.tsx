import { useNavigate } from "@tanstack/react-router";

import type { TargetsProfile } from "../api/gen/model";
import { updateProfile } from "../api/gen/query/targets/targets";
import { toast } from "../ui/toast";
import { profileSections } from "./editor";
import { useImpactSheet } from "./impact-sheet";
import {
  type ProfileDraft,
  profileBodyOf,
  profileDraftOf,
  profileProblemsOf,
} from "./profile-draft";
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

/** A saved Profile's editor, whose Save goes through the Impact. */
export function SavedProfileEditor({ profile }: { profile: TargetsProfile }) {
  const navigate = useNavigate();
  const impact = useImpactSheet();
  return (
    <>
      <ProfileEditor
        initial={profileDraftOf(profile)}
        save={{
          label: "Save",
          onSave: async (draft) => {
            const body = profileBodyOf(draft);
            await impact.save({
              title: `Save ${profile.name}`,
              action: "Save",
              run: (changing) => updateProfile(profile.id, body, changing),
              done: async () => {
                toast(`Saved ${body.name}.`);
                await navigate({ to: "/settings/$section", params: { section: "profiles" } });
              },
            });
          },
        }}
      />
      {impact.sheet}
    </>
  );
}
