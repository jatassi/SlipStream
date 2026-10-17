import { useListQualitiesSuspense } from "../api/gen/query/targets/targets";
import { Checkbox } from "../ui/checkbox";
import { RankedList } from "../ui/ranked-list";
import { type TargetDraft, accepting } from "./draft";
import type { SectionProps } from "./editor";
import { CinemaSources } from "./cinema";
import { Meaning } from "./meaning";
import { type QualityKey, cam, keepCutoff } from "./order";
import { PresetMenu } from "./presets";
import { useQualityNames } from "./quality-names";

function stateOf(draft: TargetDraft, key: QualityKey, tiedAbove: boolean): string {
  if (!draft.accepted.has(key)) {
    return "Not accepted.";
  }
  return tiedAbove ? "Tied with the one above." : "Accepted.";
}

export function QualitiesSection({ draft, onChange }: SectionProps<TargetDraft>) {
  const { qualities } = useListQualitiesSuspense().data;
  const name = useQualityNames();
  const cinema = qualities.find((quality) => quality.key === cam);
  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-ink-2">
          Tick the qualities this Target accepts and rank them. Higher is better.
        </p>
        <PresetMenu draft={draft} onChange={onChange} />
      </div>
      <RankedList
        label="Qualities"
        caption="A release that ranks above the file a Target holds replaces it, until the file reaches the cutoff."
        rows={draft.order}
        ties
        name={name}
        onChange={(order) => {
          onChange({ ...draft, order: keepCutoff(order, draft.accepted) });
        }}
        renderRow={(key) => {
          const row = draft.order.find((candidate) => candidate.id === key);
          return (
            <div className="flex flex-col">
              <Checkbox
                label={name(key)}
                checked={draft.accepted.has(key)}
                onCheckedChange={(on) => {
                  onChange(accepting(draft, key, on));
                }}
              />
              <span className="-mt-1 pl-6.5 text-meta text-ink-3 in-data-[density=touch]:pl-8.5">
                {stateOf(draft, key, row?.kind === "item" && row.tiedAbove)}
              </span>
            </div>
          );
        }}
      />
      <Meaning draft={draft} />
      {cinema !== undefined && <CinemaSources draft={draft} onChange={onChange} quality={cinema} />}
    </div>
  );
}
