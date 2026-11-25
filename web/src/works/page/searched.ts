import type { WorksContentType } from "../../api/gen/model";
import type { Searched } from "../../grabs/search";

/** What a press of Search on a Work names in its toast: a one-Item Work's label, else its Item noun. */
export function searchedOf(type: WorksContentType, label: string): Searched {
  return type.items === "one" ? { name: label } : { item: type.item };
}
