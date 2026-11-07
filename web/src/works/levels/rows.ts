import type { Fold } from "./fold";
import type { ItemView, LevelView } from "./model";

/** One line of the levels' flat list. An open level paints its card across its header, body and footer rows. */
export type Row =
  | {
      readonly kind: "header";
      readonly key: string;
      readonly level: LevelView;
      readonly open: boolean;
    }
  | {
      readonly kind: "item";
      readonly key: string;
      readonly level: LevelView;
      readonly item: ItemView;
    }
  | {
      readonly kind: "fold";
      readonly key: string;
      readonly level: LevelView;
      readonly fold: Fold<ItemView>;
    }
  | {
      readonly kind: "withdrawn";
      readonly key: string;
      readonly level: LevelView;
      readonly item: ItemView;
    }
  | { readonly kind: "footer"; readonly key: string; readonly level: LevelView };

/** The levels as rows: a closed level is its header alone, an open one its header, Items, folds, withdrawn Items and footer. */
export function levelRows(levels: readonly LevelView[], isOpen: (level: LevelView) => boolean) {
  const rows: Row[] = [];
  for (const level of levels) {
    const { id } = level;
    const open = isOpen(level);
    rows.push({ kind: "header", key: `${id} header`, level, open });
    if (!open) {
      continue;
    }
    for (const entry of level.entries) {
      rows.push(
        entry.kind === "item"
          ? { kind: "item", key: entry.item.item.id, level, item: entry.item }
          : {
              kind: "fold",
              key: `${entry.fold.items[0]?.item.id ?? id} fold`,
              level,
              fold: entry.fold,
            },
      );
    }
    for (const item of level.gone) {
      rows.push({ kind: "withdrawn", key: item.item.id, level, item });
    }
    rows.push({ kind: "footer", key: `${id} footer`, level });
  }
  return rows;
}
