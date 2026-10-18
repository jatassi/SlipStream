import {
  ArrowRight02Icon,
  CancelCircleIcon,
  CheckmarkCircle02Icon,
} from "@hugeicons/core-free-icons";

import { useListQualitiesSuspense } from "../api/gen/query/targets/targets";
import { Icon } from "../ui/icon";
import { type TargetDraft, layoutOfDraft } from "./draft";
import { useQualityNames } from "./quality-names";
import { type Sample, upgradeSentence } from "./upgrade-sentence";

function SampleRow({ sample, name }: { sample: Sample; name: (key: string) => string }) {
  return (
    <li className="flex flex-wrap items-center gap-x-3 gap-y-0.5 px-4 py-2">
      <span className="flex items-center gap-1.5 text-ink-1">
        {name(sample.from)}
        <Icon icon={ArrowRight02Icon} className="size-3.5 text-ink-3" />
        <span className="sr-only">to</span>
        {name(sample.to)}
      </span>
      {sample.allowed ? (
        <span className="flex items-center gap-1 font-medium text-ink-1">
          <Icon icon={CheckmarkCircle02Icon} className="size-4 text-ok" />
          Upgrades
        </span>
      ) : (
        <span className="flex items-center gap-1 font-medium text-ink-2">
          <Icon icon={CancelCircleIcon} className="size-4 text-ink-3" />
          Doesn&apos;t upgrade
        </span>
      )}
      <span className="basis-full text-meta text-ink-3">{sample.reason}</span>
    </li>
  );
}

export function Meaning({ draft }: { draft: TargetDraft }) {
  const { qualities } = useListQualitiesSuspense().data;
  const name = useQualityNames();
  const { sentence, samples } = upgradeSentence(
    layoutOfDraft(draft),
    qualities.map((quality) => quality.key),
    name,
  );
  return (
    <div className="flex flex-col gap-3">
      <p aria-live="polite" className="text-ink-1">
        {sentence}
      </p>
      {samples.length > 0 && (
        <ul
          aria-label="Sample moves"
          className="divide-y divide-hairline rounded-card bg-glass ring-1 ring-hairline ring-inset"
        >
          {samples.map((sample) => (
            <SampleRow key={`${sample.from}>${sample.to}`} sample={sample} name={name} />
          ))}
        </ul>
      )}
    </div>
  );
}
