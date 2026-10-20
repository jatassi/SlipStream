import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import { useListCutsSuspense } from "../api/gen/query/targets/targets";
import { useListWorksSuspense } from "../api/gen/query/works/works";
import { useQualityNames } from "../targets/quality-names";

export type Names = {
  readonly work: (id: string | undefined) => string | undefined;
  readonly node: (id: string) => string;
  readonly quality: (key: string) => string;
  readonly edition: (key: string) => string | undefined;
};

const anyEdition = "any";

export function useNames(): Names {
  const works = new Map(useListWorksSuspense().data.works.map((work) => [work.id, work.label]));
  const nodes = new Map(useListNodesSuspense().data.nodes.map((node) => [node.id, node.name]));
  const cuts = new Map(useListCutsSuspense().data.cuts.map((cut) => [cut.key, cut.name]));
  const quality = useQualityNames();
  return {
    work: (id) => (id === undefined ? undefined : works.get(id)),
    node: (id) => nodes.get(id) ?? "its Node",
    quality,
    edition: (key) => (key === anyEdition ? undefined : (cuts.get(key) ?? key)),
  };
}
