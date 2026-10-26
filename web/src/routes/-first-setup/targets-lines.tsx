import type { ReactNode } from "react";

import type {
  TargetsProfile,
  TargetsQualityList,
  TargetsSize,
  TargetsTarget,
} from "../../api/gen/model";
import { draftOf, layoutOfDraft } from "../../targets/draft";
import { useQualityNames } from "../../targets/quality-names";
import { upgradeSentence } from "../../targets/upgrade-sentence";
import { Button } from "../../ui/button";
import { Details } from "../../ui/details";

type Name = (key: string) => string;

const sizeWords = ({ min, max }: TargetsSize): string =>
  max === null
    ? `${String(min)} MiB per minute or more`
    : `${String(min)} to ${String(max)} MiB per minute`;

function Machinery({
  profile,
  target,
  name,
}: {
  profile: TargetsProfile;
  target: TargetsTarget;
  name: Name;
}) {
  const qualities = target.tiers.flatMap((tier) => tier ?? []);
  const rows: [string, ReactNode][] = [
    ["Profile", profile.name],
    ["Qualities", qualities.map((key) => name(key)).join(", ")],
    ["Cutoff", name(target.cutoff)],
    [
      "Sizes",
      <ul key="sizes">
        {target.sizes.map((size) => (
          <li key={size.quality}>
            {name(size.quality)}, {sizeWords(size)}
          </li>
        ))}
      </ul>,
    ],
  ];
  return (
    <dl className="mt-2 flex flex-col gap-2">
      {rows.map(([term, value]) => (
        <div key={term} className="flex flex-col">
          <dt className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">{term}</dt>
          <dd className="text-ink-1">{value}</dd>
        </div>
      ))}
    </dl>
  );
}

export function TargetLine({
  profile,
  target,
  qualities,
  named,
  onEdit,
}: {
  profile: TargetsProfile;
  target: TargetsTarget;
  qualities: TargetsQualityList;
  named: boolean;
  onEdit: () => void;
}) {
  const name = useQualityNames();
  const { sentence } = upgradeSentence(
    layoutOfDraft(draftOf(target, qualities)),
    qualities.qualities.map((quality) => quality.key),
    name,
  );
  return (
    <Details
      summary={named ? `${target.name}: ${sentence}` : sentence}
      action={
        <Button size="small" onClick={onEdit}>
          Edit {target.name}
        </Button>
      }
    >
      <Machinery profile={profile} target={target} name={name} />
    </Details>
  );
}
