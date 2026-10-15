import { useQuery } from "@tanstack/react-query";

import { getListWorksQueryKey, listWorks } from "../../api/gen/query/works/works";
import { matches } from "./match";

export type FoundWork = {
  readonly id: string;
  readonly label: string;
  readonly contentType: string;
};

export type PaletteProvider = { readonly useWorks: (query: string) => readonly FoundWork[] };

const shownAtRest = 5;
const shownWhileTyping = 8;

export const libraryProvider: PaletteProvider = {
  useWorks: (query) => {
    const works =
      useQuery({
        queryKey: getListWorksQueryKey(),
        queryFn: ({ signal }) => listWorks(undefined, { signal }),
      }).data?.works ?? [];
    return works
      .filter((work) => matches(query, work.label, work.sortLabel))
      .slice(0, query.trim() === "" ? shownAtRest : shownWhileTyping);
  },
};
