import type { WorksContentType } from "../api/gen/model";
import { useListRootsSuspense } from "../api/gen/query/roots/roots";
import { useListProfilesSuspense } from "../api/gen/query/targets/targets";
import { useNow } from "../tasks/use-now";
import { arrange, scopeOf } from "./arrange";
import { usePicking } from "./picking";
import { useProjection } from "./projection";
import { useMatched } from "./search";
import type { ArrangeContext, LibraryView } from "./view";

const hourMs = 3_600_000;

function useArrangeContext(view: LibraryView, types: readonly WorksContentType[]): ArrangeContext {
  const { profiles } = useListProfilesSuspense().data;
  const { roots } = useListRootsSuspense().data;
  return {
    today: useNow(hourMs),
    profiles: new Map(profiles.map(({ id, name }) => [id, name])),
    roots: new Map(roots.map(({ id, path }) => [id, path])),
    matched: useMatched(view.q, types),
  };
}

/** A Library's Works arranged by its view, with what it can offer and what is picked. */
export function useArranged(types: readonly WorksContentType[], view: LibraryView) {
  const projection = useProjection();
  const ctx = useArrangeContext(view, types);
  const scope = scopeOf(types);
  const arranged = arrange(projection.works, scope, view, ctx);
  const typeIds = new Set(types.map(({ id }) => id));
  return {
    ctx,
    scope,
    arranged,
    works: projection.works.filter((work) => typeIds.has(work.type.id)),
    picking: usePicking(projection.byId, arranged.order),
  };
}
