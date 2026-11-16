import { bytes } from "../format";
import { type SizeDraft, type TargetDraft, acceptedInOrder, sizeOf, sizeProblem } from "./draft";
import type { SectionProps } from "./editor";
import { DraftField } from "./field";
import { useQualityNames } from "./quality-names";

const twoHours = 120;
const mib = 1 << 20;

function filmSize({ min, max }: SizeDraft): string | undefined {
  if (sizeProblem({ min, max }) !== undefined) {
    return undefined;
  }
  const low = bytes(Number(min) * twoHours * mib);
  return max.trim() === ""
    ? `A two-hour film takes at least ${low}.`
    : `A two-hour film takes ${low} to ${bytes(Number(max) * twoHours * mib)}.`;
}

function SizeRow({
  name,
  size,
  onSize,
}: {
  name: string;
  size: SizeDraft;
  onSize: (size: SizeDraft) => void;
}) {
  const problem = sizeProblem(size);
  return (
    <li className="grid grid-cols-2 items-start gap-x-3 gap-y-2 px-4 py-3 @min-[620px]:grid-cols-[minmax(9rem,1fr)_10rem_10rem_minmax(14rem,1.5fr)]">
      <span className="col-span-2 font-semibold text-ink-1 @min-[620px]:col-span-1 @min-[620px]:pt-7">
        {name}
      </span>
      <div className="contents">
        <DraftField
          label="Minimum"
          name={`${name} minimum`}
          inputMode="decimal"
          value={size.min}
          error={problem?.min}
          onChange={(min) => {
            onSize({ ...size, min });
          }}
        />
        <DraftField
          label="Maximum"
          name={`${name} maximum`}
          inputMode="decimal"
          value={size.max}
          placeholder="No limit"
          error={problem?.max}
          onChange={(max) => {
            onSize({ ...size, max });
          }}
        />
      </div>
      <p className="text-meta text-ink-3">
        {filmSize(size) ?? "Fix the size to see what it means."}
      </p>
    </li>
  );
}

export function SizesSection({ draft, onChange }: SectionProps<TargetDraft>) {
  const name = useQualityNames();
  const accepted = acceptedInOrder(draft);
  return (
    <div className="flex flex-col gap-3">
      <p className="text-ink-2">
        In MiB per minute of runtime. A release outside its range isn&apos;t grabbed automatically.
        When a Work&apos;s runtime is unknown, SlipStream skips the check and notes that on the
        release.
      </p>
      {accepted.length === 0 ? (
        <p className="text-ink-3">Accept a quality to set its size.</p>
      ) : (
        <ul
          aria-label="Sizes"
          className="@container divide-y divide-hairline rounded-card bg-glass ring-1 ring-hairline ring-inset"
        >
          {accepted.map((key) => (
            <SizeRow
              key={key}
              name={name(key)}
              size={sizeOf(draft, key)}
              onSize={(size) => {
                onChange({ ...draft, sizes: { ...draft.sizes, [key]: size } });
              }}
            />
          ))}
        </ul>
      )}
    </div>
  );
}
