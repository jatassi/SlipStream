import type { ComponentType } from "react";

import type { TargetDraft } from "./draft";
import { FiltersSection } from "./filters";
import { NameSection } from "./name-section";
import { PropersSection } from "./propers-section";
import { QualitiesSection } from "./qualities";
import { ScoresSection } from "./scores-section";
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
  { id: "scores", label: "Scores", component: ScoresSection },
  { id: "propers", label: "Propers and repacks", component: PropersSection },
];
