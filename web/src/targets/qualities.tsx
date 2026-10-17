import { useListQualitiesSuspense } from "../api/gen/query/targets/targets";
import type { TargetDraft } from "./draft";
import type { SectionProps } from "./editor";
import { CinemaSources } from "./cinema";
import { Meaning } from "./meaning";
import { cam } from "./order";
import { PresetMenu } from "./presets";
import { QualityLists } from "./quality-lists";

export function QualitiesSection({ draft, onChange }: SectionProps<TargetDraft>) {
  const { qualities } = useListQualitiesSuspense().data;
  const cinema = qualities.find((quality) => quality.key === cam);
  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-ink-2">
          Tick the qualities this Target accepts and rank them. Higher is better.
        </p>
        <PresetMenu draft={draft} onChange={onChange} />
      </div>
      <QualityLists draft={draft} onChange={onChange} />
      <Meaning draft={draft} />
      {cinema !== undefined && <CinemaSources draft={draft} onChange={onChange} quality={cinema} />}
    </div>
  );
}
