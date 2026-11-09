import type { EditorSection } from "./editor";
import type { ProfileDraft } from "./profile-draft";
import { ProfileEditionsSection } from "./profile-editions";
import { ProfileNameSection } from "./profile-name";
import { ProfileTargetsSection } from "./profile-targets";

export const profileSections: readonly EditorSection<ProfileDraft>[] = [
  { id: "name", label: "Name", component: ProfileNameSection },
  { id: "targets", label: "Targets", component: ProfileTargetsSection },
  { id: "editions", label: "Editions", component: ProfileEditionsSection },
];
