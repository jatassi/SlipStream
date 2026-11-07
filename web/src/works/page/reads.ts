import type { WorksWorkView } from "../../api/gen/model";
import { useGetWorkCellsSuspense, useListWorkFilesSuspense } from "../../api/gen/query/files/files";
import { useListGrabsSuspense } from "../../api/gen/query/grabs/grabs";
import {
  useGetWorkWantsSuspense,
  useListCutsSuspense,
  useListProfilesSuspense,
  useListTargetsSuspense,
} from "../../api/gen/query/targets/targets";
import { cutNameOf, matrixOf } from "./matrix-model";

/** What the Work page's sections read besides the Work itself, with the names its cells use. */
export function useWorkReads(work: WorksWorkView) {
  const cells = useGetWorkCellsSuspense(work.id).data;
  const wants = useGetWorkWantsSuspense(work.id).data;
  const listing = useListWorkFilesSuspense(work.id).data;
  const { cuts } = useListCutsSuspense().data;
  const { targets } = useListTargetsSuspense().data;
  const profile = useListProfilesSuspense().data.profiles.find(({ id }) => id === work.profile.id);
  const grabs = useListGrabsSuspense().data.grabs.filter(({ workId }) => workId === work.id);
  if (profile === undefined) {
    throw new Error(`the Work's Profile ${work.profile.id} isn't listed`);
  }
  return {
    cells,
    matrix: matrixOf({ cells: cells.cells, wants, profile, targets, cuts }),
    wants,
    listing,
    cuts,
    targets,
    profile,
    grabs,
    cutName: cutNameOf(cuts),
    targetName: (id: string) => targets.find((target) => target.id === id)?.name ?? id,
  };
}

export type WorkReads = ReturnType<typeof useWorkReads>;
