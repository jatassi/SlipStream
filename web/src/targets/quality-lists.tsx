import { useEffect, useRef } from "react";

import { useListQualitiesSuspense } from "../api/gen/query/targets/targets";
import { Checkbox } from "../ui/checkbox";
import { List, ListRow } from "../ui/list";
import { RankedList } from "../ui/ranked-list";
import { type TargetDraft, accepting, unacceptedOf } from "./draft";
import type { SectionProps } from "./editor";
import { type QualityKey, keepCutoff, rankedOf } from "./order";
import { useQualityNames } from "./quality-names";

// Ticking a quality moves it between the two lists, so its checkbox mounts
// again elsewhere; focus follows it there.
function useFollowFocus() {
  const section = useRef<HTMLDivElement>(null);
  const moved = useRef<QualityKey | null>(null);
  useEffect(() => {
    const key = moved.current;
    moved.current = null;
    if (key !== null) {
      section.current
        ?.querySelector<HTMLElement>(`[data-quality="${CSS.escape(key)}"] [role="checkbox"]`)
        ?.focus();
    }
  });
  return {
    section,
    follow: (key: QualityKey) => {
      moved.current = key;
    },
  };
}

function Unaccepted({
  keys,
  name,
  tick,
}: {
  keys: readonly QualityKey[];
  name: (key: QualityKey) => string;
  tick: (key: QualityKey) => (on: boolean) => void;
}) {
  if (keys.length === 0) {
    return null;
  }
  return (
    <div className="flex flex-col gap-2">
      <h3 className="text-body font-semibold text-ink-2">Not accepted</h3>
      <List label="Not accepted">
        {keys.map((key) => (
          <ListRow
            key={key}
            title={
              <span data-quality={key} className="flex">
                <Checkbox label={name(key)} checked={false} onCheckedChange={tick(key)} />
              </span>
            }
          />
        ))}
      </List>
    </div>
  );
}

export function QualityLists({ draft, onChange }: SectionProps<TargetDraft>) {
  const { qualities, presets } = useListQualitiesSuspense().data;
  const name = useQualityNames();
  const { section, follow } = useFollowFocus();
  const unaccepted = unacceptedOf(
    draft,
    qualities.map((quality) => quality.key),
  );
  const tick = (key: QualityKey) => (on: boolean) => {
    follow(key);
    onChange(accepting(draft, key, on, presets[0]?.tiers ?? []));
  };
  return (
    <div ref={section} className="flex flex-col gap-5">
      {rankedOf(draft.order).length === 0 ? (
        <p className="rounded-card bg-glass px-4 py-3 text-ink-2 ring-1 ring-hairline ring-inset">
          This Target accepts no quality yet. Tick one below to rank it.
        </p>
      ) : (
        <RankedList
          label="Qualities"
          caption="A release that ranks above the file a Target holds replaces it, until the file reaches the cutoff."
          rows={draft.order}
          ties
          name={name}
          onChange={(order) => {
            onChange({ ...draft, order: keepCutoff(order) });
          }}
          renderRow={(key) => (
            <div data-quality={key} className="flex">
              <Checkbox label={name(key)} checked onCheckedChange={tick(key)} />
            </div>
          )}
        />
      )}
      <Unaccepted keys={unaccepted} name={name} tick={tick} />
    </div>
  );
}
