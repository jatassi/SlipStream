import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import { useListCutsSuspense } from "../api/gen/query/targets/targets";
import { useProjection } from "../library/projection";
import { type WorkNames, libraryNames } from "./remove-preview";

export type Names = {
  readonly work: (id: string | undefined) => string | undefined;
  readonly works: WorkNames;
  readonly poster: (id: string | undefined) => string | undefined;
  readonly node: (id: string) => string;
  readonly edition: (key: string) => string | undefined;
};

const anyEdition = "any";

export function useNames(): Names {
  const works = useProjection().byId;
  const nodes = new Map(useListNodesSuspense().data.nodes.map((node) => [node.id, node.name]));
  const cuts = new Map(useListCutsSuspense().data.cuts.map((cut) => [cut.key, cut.name]));
  const work = (id: string | undefined) => (id === undefined ? undefined : works.get(id)?.label);
  return {
    work,
    works: libraryNames(work),
    poster: (id) => (id === undefined ? undefined : works.get(id)?.poster),
    node: (id) => nodes.get(id) ?? "its Node",
    edition: (key) => (key === anyEdition ? undefined : (cuts.get(key) ?? key)),
  };
}
