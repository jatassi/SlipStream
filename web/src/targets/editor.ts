import type { ComponentType } from "react";

import type { TargetDraft } from "./draft";
import { FiltersSection } from "./filters";
import { NameSection } from "./name-section";
import type { ProfileDraft } from "./profile-draft";
import { ProfileEditionsSection } from "./profile-editions";
import { ProfileNameSection } from "./profile-name";
import { ProfileTargetsSection } from "./profile-targets";
import { QualitiesSection } from "./qualities";
import { SizesSection } from "./sizes";

export type SectionProps<Draft> = {
  readonly draft: Draft;
  readonly onChange: (next: Draft) => void;
};

// A later spec adds a section as a row here and a field on its draft, and
// touches no editor.
export type EditorSection<Draft> = {
  readonly id: string;
  readonly label: string;
  readonly component: ComponentType<SectionProps<Draft>>;
};

export const targetSections: readonly EditorSection<TargetDraft>[] = [
  { id: "name", label: "Name", component: NameSection },
  { id: "qualities", label: "Qualities", component: QualitiesSection },
  { id: "filters", label: "Filters", component: FiltersSection },
  { id: "sizes", label: "Sizes", component: SizesSection },
];

export const profileSections: readonly EditorSection<ProfileDraft>[] = [
  { id: "name", label: "Name", component: ProfileNameSection },
  { id: "targets", label: "Targets", component: ProfileTargetsSection },
  { id: "editions", label: "Editions", component: ProfileEditionsSection },
];
