import { useNavigate } from "@tanstack/react-router";

import type { TargetsProfile } from "../api/gen/model";
import { createProfile } from "../api/gen/query/targets/targets";
import { toast } from "../ui/toast";
import { profileSections } from "./profile-sections";
import { savingCuts } from "./editions-draft";
import { useProfileSave } from "./use-profile-save";
import {
  type ProfileDraft,
  newProfileDraft,
  profileBodyOf,
  profileDraftOf,
  profileProblemsOf,
} from "./profile-draft";
import { type Save, SectionEditor } from "./section-editor";

function ProfileEditor({ initial, save }: { initial: ProfileDraft; save: Save<ProfileDraft> }) {
  return (
    <SectionEditor
      sections={profileSections}
      initial={initial}
      problemsOf={profileProblemsOf}
      save={save}
    />
  );
}

export function NewProfileEditor({ contentType }: { contentType: string }) {
  const navigate = useNavigate();
  return (
    <ProfileEditor
      initial={newProfileDraft(contentType)}
      save={{
        label: "Create Profile",
        onSave: async (draft) => {
          const made = await savingCuts(() => createProfile(profileBodyOf(draft)));
          toast(`Created ${made.name}.`);
          await navigate({ to: "/settings/$section", params: { section: "profiles" } });
        },
      }}
    />
  );
}

export function SavedProfileEditor({ profile }: { profile: TargetsProfile }) {
  const navigate = useNavigate();
  const saving = useProfileSave(profile, async () => {
    await navigate({ to: "/settings/$section", params: { section: "profiles" } });
  });
  return (
    <>
      <ProfileEditor
        initial={profileDraftOf(profile)}
        save={{ label: "Save", onSave: saving.save }}
      />
      {saving.dialogs}
    </>
  );
}
