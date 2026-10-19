import { useQuery } from "@tanstack/react-query";

import type { CheckFolderInputBodyPurpose } from "../api/gen/model";
import { checkFolder } from "../api/gen/query/roots/roots";
import { type FolderCheck, type FolderVerdict, verdictKey } from "../folders/verdict";
import { type Namer, verdictOf } from "./words";

export type RootVerdict = FolderVerdict & { readonly network: boolean };

export type RootCheck = (...args: Parameters<FolderCheck>) => Promise<RootVerdict>;

/** The library root rules for a folder, judged on its Node; rootId is the root being re-homed. */
export const rootCheck =
  (namer: Namer, rootId?: string): RootCheck =>
  async ({ nodeId, path, signal }) => {
    const verdict = await checkFolder(
      nodeId,
      { path, purpose: "root", ...(rootId !== undefined && { rootId }) },
      { signal },
    );
    return { ...verdictOf(verdict, namer), network: verdict.volume?.kind === "network" };
  };

/** A purpose's rules for a folder, judged on its Node. */
export const purposeCheck =
  (purpose: CheckFolderInputBodyPurpose, namer: Namer): FolderCheck =>
  async ({ nodeId, path, signal }) =>
    verdictOf(await checkFolder(nodeId, { path, purpose }, { signal }), namer);

/** The verdict FolderField shows for the same folder, read from the one request. */
export function useRootVerdict(check: RootCheck, nodeId: string, path: string) {
  return useQuery({
    queryKey: verdictKey(nodeId, path),
    queryFn: ({ signal }) => check({ nodeId, path, signal }),
    enabled: path !== "",
    staleTime: 0,
  }).data;
}
