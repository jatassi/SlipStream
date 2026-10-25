import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import { useListCutsSuspense } from "../api/gen/query/targets/targets";
import { useListContentTypesSuspense, useListWorksSuspense } from "../api/gen/query/works/works";
import { useQualityNames } from "../targets/quality-names";

export type Names = {
  readonly work: (id: string | undefined) => string | undefined;
  readonly poster: (id: string | undefined) => string | undefined;
  readonly node: (id: string) => string;
  readonly quality: (key: string) => string;
  readonly edition: (key: string) => string | undefined;
};

const anyEdition = "any";

export function useNames(): Names {
  const works = new Map(useListWorksSuspense().data.works.map((work) => [work.id, work]));
  const posterKinds = new Set(
    useListContentTypesSuspense().data.contentTypes.flatMap((type) =>
      type.artwork
        .filter((kind) => kind.shape === "poster" && kind.tier === "work")
        .map((kind) => `${type.id}/${kind.key}`),
    ),
  );
  const nodes = new Map(useListNodesSuspense().data.nodes.map((node) => [node.id, node.name]));
  const cuts = new Map(useListCutsSuspense().data.cuts.map((cut) => [cut.key, cut.name]));
  const quality = useQualityNames();
  return {
    work: (id) => (id === undefined ? undefined : works.get(id)?.label),
    poster: (id) => {
      const work = id === undefined ? undefined : works.get(id);
      return work?.artwork.find((art) => posterKinds.has(`${work.contentType}/${art.kind}`))?.url;
    },
    node: (id) => nodes.get(id) ?? "its Node",
    quality,
    edition: (key) => (key === anyEdition ? undefined : (cuts.get(key) ?? key)),
  };
}
