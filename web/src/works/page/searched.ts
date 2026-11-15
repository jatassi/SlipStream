import type { FilesCell, WorksContentType } from "../../api/gen/model";
import type { Searched } from "../../grabs/search";

/**
 * What a press of Search on a Work, or on the Items `among` keeps, names in
 * its toast: a one-Item Work's label, else a count of its Items with a
 * wanted cell.
 */
export function searchedOf(
  type: WorksContentType,
  label: string,
  cells: readonly FilesCell[],
  among: (itemId: string) => boolean = () => true,
): Searched {
  if (type.items === "one") {
    return { name: label };
  }
  const items = new Set(cells.filter((c) => c.wanted && among(c.itemId)).map((c) => c.itemId));
  return { count: items.size, item: type.item };
}
