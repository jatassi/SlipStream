import type { TargetsQualityKey } from "../api/gen/model";
import type { RankedRow } from "../ui/ranked-list";

export type QualityKey = TargetsQualityKey;

export type QualityRow = RankedRow<QualityKey>;

/** The ranked list's rows: every quality but CAM, and the cutoff marker. */
export type Order = readonly QualityRow[];

export const cam: QualityKey = "cam";

export const cutoffMarker: QualityRow = { kind: "marker", id: "cutoff", label: "Cutoff" };

/** A Target's accepted qualities as the API writes them: tiers best first and the cutoff tier. */
export type Layout = {
  readonly tiers: readonly (readonly QualityKey[])[];
  readonly cutoff: number;
};

const item = (value: QualityKey, tiedAbove: boolean): QualityRow => ({
  kind: "item",
  id: value,
  value,
  tiedAbove,
});

const rowsOfTiers = (tiers: readonly (readonly QualityKey[])[]): QualityRow[] =>
  tiers.flatMap((tier) => tier.map((value, at) => item(value, at > 0)));

// A unit is what the marker can sit between: an item and the items tied below it.
const tiedAbove = (row: QualityRow | undefined) => row?.kind === "item" && row.tiedAbove;

function unitEnd(rows: Order, at: number): number {
  let end = at + 1;
  while (tiedAbove(rows[end])) {
    end += 1;
  }
  return end;
}

const isAccepted = (accepted: ReadonlySet<QualityKey>, row: QualityRow) =>
  row.kind === "item" && accepted.has(row.value);

/** The rows with the cutoff placed just below the first or last tier holding an accepted quality. */
export function placeCutoff(
  rows: Order,
  accepted: ReadonlySet<QualityKey>,
  at: "top" | "bottom",
): QualityRow[] {
  const items = rows.filter((row) => row.kind === "item");
  const first = items.findIndex((row) => isAccepted(accepted, row));
  if (first === -1) {
    return [cutoffMarker, ...items];
  }
  const holder = at === "top" ? first : items.findLastIndex((row) => isAccepted(accepted, row));
  let start = holder;
  while (start > 0 && tiedAbove(items[start])) {
    start -= 1;
  }
  const end = unitEnd(items, start);
  return [...items.slice(0, end), cutoffMarker, ...items.slice(end)];
}

/**
 * Keeps the cutoff in range: an accepted quality must rank above it, so a
 * marker left above every accepted quality moves just below the first.
 */
export function keepCutoff(rows: Order, accepted: ReadonlySet<QualityKey>): QualityRow[] {
  const marker = rows.findIndex((row) => row.kind === "marker");
  return rows.slice(0, marker).some((row) => isAccepted(accepted, row))
    ? [...rows]
    : placeCutoff(rows, accepted, "top");
}

/** The rows a preset lays out, CAM left out, with the cutoff at the top accepted tier. */
export const presetOrder = (
  tiers: readonly (readonly string[] | null)[],
  accepted: ReadonlySet<QualityKey>,
  keys: readonly QualityKey[],
): QualityRow[] =>
  placeCutoff(
    rowsOfTiers(
      tiers.map((tier) =>
        (tier ?? []).flatMap((key) => keys.find((known) => known === key && known !== cam) ?? []),
      ),
    ),
    accepted,
    "top",
  );

/**
 * A saved Target's rows: its tiers with the cutoff below its cutoff tier,
 * then every quality it doesn't accept, each alone, in `rest`'s order.
 */
export function orderOf(layout: Layout, rest: readonly QualityKey[]): QualityRow[] {
  const tiers = layout.tiers.map((tier) => tier.filter((key) => key !== cam));
  const above = tiers.slice(0, layout.cutoff + 1).filter((tier) => tier.length > 0);
  const below = tiers.slice(layout.cutoff + 1).filter((tier) => tier.length > 0);
  const held = new Set(tiers.flat());
  const others = rest.filter((key) => key !== cam && !held.has(key)).map((key) => [key]);
  return [...rowsOfTiers(above), cutoffMarker, ...rowsOfTiers([...below, ...others])];
}

/**
 * The accepted qualities in tiers, best first, with CAM alone at the
 * bottom when accepted, and the index of the tier just above the cutoff.
 */
export function layoutOf(rows: Order, accepted: ReadonlySet<QualityKey>): Layout {
  const tiers: QualityKey[][] = [];
  let tier: QualityKey[] = [];
  let cutoff = -1;
  const close = () => {
    if (tier.length > 0) {
      tiers.push(tier);
    }
    tier = [];
  };
  for (const row of rows) {
    if (row.kind === "marker") {
      close();
      cutoff = tiers.length - 1;
      continue;
    }
    if (!row.tiedAbove) {
      close();
    }
    if (accepted.has(row.value)) {
      tier.push(row.value);
    }
  }
  close();
  if (accepted.has(cam)) {
    tiers.push([cam]);
  }
  return { tiers, cutoff: Math.max(cutoff, 0) };
}
