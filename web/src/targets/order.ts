import type { TargetsQualityKey } from "../api/gen/model";
import { type RankedRow, settle } from "../ui/ranked-list";

export type QualityKey = TargetsQualityKey;

export type QualityRow = RankedRow<QualityKey>;

/** The ranked list's rows: the accepted qualities but CAM, in tiers best first, and the cutoff marker. */
export type Order = readonly QualityRow[];

export type Tiers = readonly (readonly QualityKey[])[];

/** A preset's tiers as the API lists them. */
export type PresetTiers = readonly (readonly string[] | null)[];

export const cam: QualityKey = "cam";

export const cutoffMarker: QualityRow = { kind: "marker", id: "cutoff", label: "Cutoff" };

/** A Target's accepted qualities as the API writes them: tiers best first and the cutoff tier. */
export type Layout = {
  readonly tiers: Tiers;
  readonly cutoff: number;
};

const item = (value: QualityKey, tiedAbove: boolean): QualityRow => ({
  kind: "item",
  id: value,
  value,
  tiedAbove,
});

const rowsOfTiers = (tiers: Tiers): QualityRow[] =>
  tiers.flatMap((tier) => tier.map((value, at) => item(value, at > 0)));

const tiedAbove = (row: QualityRow | undefined) => row?.kind === "item" && row.tiedAbove;

function tierStart(rows: Order, at: number): number {
  let start = at;
  while (tiedAbove(rows[start])) {
    start -= 1;
  }
  return start;
}

function tierEnd(rows: Order, at: number): number {
  let end = at + 1;
  while (tiedAbove(rows[end])) {
    end += 1;
  }
  return end;
}

/** The ranked qualities, without the cutoff. */
export const rankedOf = (rows: Order): QualityKey[] =>
  rows.flatMap((row) => (row.kind === "item" ? [row.value] : []));

/** The rows with the cutoff just below their first or their last tier. */
export function placeCutoff(rows: Order, at: "top" | "bottom"): QualityRow[] {
  const items = rows.filter((row) => row.kind === "item");
  const end = at === "top" ? Math.min(tierEnd(items, 0), items.length) : items.length;
  return [...items.slice(0, end), cutoffMarker, ...items.slice(end)];
}

/** Keeps the cutoff in range: at least one tier ranks above it. */
export const keepCutoff = (rows: Order): QualityRow[] =>
  rows[0]?.kind === "marker" ? placeCutoff(rows, "top") : [...rows];

/** A preset's tiers kept to the given qualities, CAM left out, with the cutoff below the top tier. */
export const presetOrder = (tiers: PresetTiers, kept: readonly QualityKey[]): QualityRow[] =>
  placeCutoff(
    rowsOfTiers(
      tiers.map((tier) =>
        (tier ?? []).flatMap((key) => kept.find((known) => known === key && known !== cam) ?? []),
      ),
    ),
    "top",
  );

/**
 * The rows with one more quality, where the preset ranks it: tied with the
 * first accepted quality it ties with there, else as a tier of its own just
 * above the first it outranks, else at the bottom. A cutoff just above that
 * tier stays above it, so accepting a quality never moves where upgrades stop.
 */
export function accept(rows: Order, key: QualityKey, preset: PresetTiers): QualityRow[] {
  const tierOf = (quality: QualityKey) =>
    preset.findIndex((tier) => (tier ?? []).includes(quality));
  const mine = tierOf(key);
  const at = rows.findIndex((row) => row.kind === "item" && tierOf(row.value) >= mine);
  const found = rows[at];
  if (found?.kind !== "item") {
    return keepCutoff([...rows, item(key, false)]);
  }
  const tied = tierOf(found.value) === mine;
  const place = tied ? tierEnd(rows, at) : tierStart(rows, at);
  return keepCutoff([...rows.slice(0, place), item(key, tied), ...rows.slice(place)]);
}

/** The rows without a quality, the tier it headed kept together and the cutoff in range. */
export const drop = (rows: Order, key: QualityKey): QualityRow[] =>
  keepCutoff(settle(rows.filter((row) => row.id !== key)));

/** A saved Target's rows: its tiers, CAM left out, with the cutoff below its cutoff tier. */
export function orderOf(layout: Layout): QualityRow[] {
  const tiers = layout.tiers.map((tier) => tier.filter((key) => key !== cam));
  const above = tiers.slice(0, layout.cutoff + 1).filter((tier) => tier.length > 0);
  const below = tiers.slice(layout.cutoff + 1).filter((tier) => tier.length > 0);
  return [...rowsOfTiers(above), cutoffMarker, ...rowsOfTiers(below)];
}

/**
 * The ranked qualities in tiers, best first, with CAM alone at the bottom
 * when accepted, and the index of the tier just above the cutoff.
 */
export function layoutOf(rows: Order, cinema: boolean): Layout {
  const tiers: QualityKey[][] = [];
  let cutoff = 0;
  for (const row of rows) {
    if (row.kind === "marker") {
      cutoff = Math.max(tiers.length - 1, 0);
    } else if (row.tiedAbove) {
      tiers.at(-1)?.push(row.value);
    } else {
      tiers.push([row.value]);
    }
  }
  if (cinema) {
    tiers.push([cam]);
  }
  return { tiers, cutoff };
}
