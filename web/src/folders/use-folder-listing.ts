import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { z } from "zod";

import { isProblem, sentenceOf } from "../api/errors";
import type { RootsFolderProblem, RootsListing } from "../api/gen/model";
import {
  browseFolders,
  getBrowseFoldersQueryKey,
  getSuggestFoldersQueryKey,
  suggestFolders,
} from "../api/gen/query/roots/roots";
import { RootsFolderProblem as FolderProblemBody } from "../api/gen/zod/slipStream.schemas.zod";
import { isPath } from "./browser-state";
import { useSettled } from "./use-settled";

/** Why the list area shows a card instead of folders. */
export type FolderProblem =
  | { readonly kind: "unreadable"; readonly problem: RootsFolderProblem }
  | { readonly kind: "offline" }
  /** Any other refusal, in the Node's own words. */
  | { readonly kind: "refused"; readonly sentence: string };

const unreadable = z.object({ problem: FolderProblemBody });

export function problemOf(error: unknown): FolderProblem {
  if (isProblem(error, "folder_unreadable")) {
    return { kind: "unreadable", problem: unreadable.parse(error.body).problem };
  }
  if (isProblem(error, "node_offline")) {
    return { kind: "offline" };
  }
  return { kind: "refused", sentence: sentenceOf(error) };
}

const skeletonAfter = 300;
const suggestAfter = 150;

export type FolderListing = {
  /** The listing to draw: the folder asked for, or the last one while it loads. */
  readonly shown: RootsListing | undefined;
  /** The listing of the folder asked for, once it has come. */
  readonly current: RootsListing | undefined;
  readonly loading: boolean;
  /** The load has run long enough that the old listing gives way to skeleton rows. */
  readonly skeleton: boolean;
  readonly problem: FolderProblem | null;
  readonly retry: () => void;
};

/**
 * One folder, or the Node's places for "", read fresh each time it is shown:
 * a folder changes on disk with no event to say so.
 */
export function useFolderListing(nodeId: string, path: string, hidden: boolean): FolderListing {
  const params = { path, hidden };
  const query = useQuery({
    queryKey: getBrowseFoldersQueryKey(nodeId, params),
    queryFn: ({ signal }) => browseFolders(nodeId, params, { signal }),
    placeholderData: keepPreviousData,
    staleTime: 0,
  });
  const loading = query.isPlaceholderData || query.isPending;
  const load = loading ? JSON.stringify([nodeId, path, hidden]) : null;
  const slow = useSettled(load, skeletonAfter, null);
  return {
    shown: query.isError ? undefined : query.data,
    current: query.isError || query.isPlaceholderData ? undefined : query.data,
    loading,
    skeleton: load !== null && slow === load,
    problem: query.isError ? problemOf(query.error) : null,
    retry: () => {
      void query.refetch();
    },
  };
}

const none: readonly string[] = [];

/** The Node's completions of a typed path, once the typing pauses. */
export function useSuggestions(nodeId: string, text: string): readonly string[] {
  const settled = useSettled(text, suggestAfter);
  const asked = isPath(settled);
  const query = useQuery({
    queryKey: getSuggestFoldersQueryKey(nodeId, { text: settled }),
    queryFn: ({ signal }) => suggestFolders(nodeId, { text: settled }, { signal }),
    enabled: asked,
    placeholderData: keepPreviousData,
    staleTime: 0,
    retry: false,
  });
  return asked && isPath(text) ? (query.data?.suggestions ?? none) : none;
}
