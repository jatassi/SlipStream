import type { ComponentType } from "react";

import type { TargetDraft } from "./draft";
import { FiltersSection } from "./filters";
import { NameSection } from "./name-section";
import { QualitiesSection } from "./qualities";
import { SizesSection } from "./sizes";

export type SectionProps<Draft> = {
  readonly draft: Draft;
  readonly onChange: (next: Draft) => void;
};

/**
 * One editor section: a heading and the control that edits its slice of the
 * draft. A later spec adds a row (Scores, Hold, Conversion settings,
 * download steps) and its draft field, and touches no editor.
 */
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
