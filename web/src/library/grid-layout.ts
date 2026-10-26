import type { Arranged } from "./arrange";
import type { Section } from "./sections";

export type GridRow =
  | { readonly kind: "section"; readonly section: Section; readonly count: number }
  | { readonly kind: "posters"; readonly ids: readonly string[]; readonly first: number };

export type GridLayout = {
  readonly rows: readonly GridRow[];
  /** The row that holds the Work at this place in the shown order. */
  readonly rowOf: (index: number) => number;
};

/** Each section's heading row, then its Works in rows of `perRow`. */
export function gridRows({ order, sections }: Pick<Arranged, "order" | "sections">, perRow: number): GridLayout {
  const rows: GridRow[] = [];
  const rowAt: number[] = [];
  for (const { section, start, count } of sections) {
    rows.push({ kind: "section", section, count });
    for (let first = start; first < start + count; first += perRow) {
      const ids = order.slice(first, Math.min(first + perRow, start + count));
      for (const _ of ids) {
        rowAt.push(rows.length);
      }
      rows.push({ kind: "posters", ids, first });
    }
  }
  return { rows, rowOf: (index) => rowAt[index] ?? -1 };
}
