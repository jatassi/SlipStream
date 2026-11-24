import type { MovesMovePreview, MovesPlace } from "../api/gen/model";
import { bytes } from "../format";
import { Details } from "../ui/details";
import { PreviewGroups } from "./preview-groups";

/** A root, or a torrent's Download Client and its Node. */
function where(place: MovesPlace): string {
  return place.client === undefined ? place.root : `${place.client} on ${place.nodeName}`;
}

/** Each pair of roots or clients the preview copies between, with the bytes that cross. */
function Machinery({ plan }: { plan: MovesMovePreview }) {
  const pairs = new Map<string, { from: string; to: string; bytes: number }>();
  for (const unit of plan.units) {
    if (unit.verdict === "red") {
      continue;
    }
    const from = where(unit.from);
    const to = where(unit.to);
    const key = `${from}>${to}`;
    const pair = pairs.get(key) ?? { from, to, bytes: 0 };
    pairs.set(key, { ...pair, bytes: pair.bytes + unit.bytes });
  }
  if (pairs.size === 0) {
    return null;
  }
  return (
    <Details>
      <ul className="mt-1 flex flex-col gap-1">
        {[...pairs].map(([key, pair]) => (
          <li key={key}>
            {`From ${pair.from} to ${pair.to}. ${pair.bytes === 0 ? "Nothing to copy, because the files are linked or cloned on one disk." : `Copies ${bytes(pair.bytes)}.`}`}
          </li>
        ))}
      </ul>
    </Details>
  );
}

/** The recommendation first, then each unit by verdict, then the copies under Details. */
export function Plan({ plan, homed }: { plan: MovesMovePreview; homed: string | undefined }) {
  return (
    <>
      <p className="text-ink-1">{plan.recommendation}</p>
      {homed !== undefined && <p>{homed}</p>}
      <PreviewGroups units={plan.units} />
      <Machinery plan={plan} />
    </>
  );
}
