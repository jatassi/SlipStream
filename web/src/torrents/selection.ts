/**
 * The Torrents picked for a bulk action. The anchor is the last one toggled
 * on its own, where a shift-click range starts.
 */
export type Picks = { readonly ids: ReadonlySet<string>; readonly anchor: string | undefined };

export const noPicks: Picks = { ids: new Set(), anchor: undefined };

/**
 * Toggles id. A range toggle sets every row from the anchor to id, in the
 * shown order, to id's new state.
 */
export function toggle(picks: Picks, id: string, order: readonly string[], range: boolean): Picks {
  const on = !picks.ids.has(id);
  const from = picks.anchor === undefined ? -1 : order.indexOf(picks.anchor);
  const to = order.indexOf(id);
  const span =
    range && from >= 0 && to >= 0 ? order.slice(Math.min(from, to), Math.max(from, to) + 1) : [id];
  const ids = new Set(picks.ids);
  for (const each of span) {
    if (on) {
      ids.add(each);
    } else {
      ids.delete(each);
    }
  }
  return { ids, anchor: id };
}

/** Every shown row picked, or none when all already are. */
export function toggleAll(picks: Picks, order: readonly string[]): Picks {
  return order.every((id) => picks.ids.has(id))
    ? noPicks
    : { ids: new Set(order), anchor: undefined };
}

/** The shown rows picked, in the shown order; a pick a filter hides doesn't count. */
export const pickedOf = <R extends { readonly id: string }>(
  picks: Picks,
  rows: readonly R[],
): R[] => rows.filter((row) => picks.ids.has(row.id));
