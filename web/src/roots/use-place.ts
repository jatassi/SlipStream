import { useSelector } from "@tanstack/react-form";
import { useState } from "react";

import { useListRootsSuspense } from "../api/gen/query/roots/roots";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { checkAfter } from "../folders/folder-field";
import { useSettled } from "../folders/use-settled";
import { useAppForm } from "../ui/form";
import { rootCheck, useRootVerdict } from "./check";
import { useStorageNodes } from "./storage-nodes";

export type Place = { readonly nodeId: string; readonly path: string };

export const samePlace = (a: Place, b: Place): boolean =>
  a.nodeId === b.nodeId && a.path === b.path;

/** A root's Node and folder as a dialog edits them, and the folder's verdict on that Node. */
export function usePlace(start: Place, rootId?: string) {
  const { roots } = useListRootsSuspense().data;
  const { contentTypes } = useListContentTypesSuspense().data;
  const { storage } = useStorageNodes();
  const [nodeId, setNodeId] = useState(
    storage.some((node) => node.id === start.nodeId) ? start.nodeId : (storage[0]?.id ?? ""),
  );
  const form = useAppForm({ defaultValues: { path: start.path } });
  const path = useSelector(form.store, (state) => state.values.path).trim();
  const settled = useSettled(path, checkAfter);
  const node = storage.find((candidate) => candidate.id === nodeId);
  const namer = { node: node?.name ?? "", roots, types: contentTypes };
  const check = rootCheck(namer, rootId);
  const verdict = useRootVerdict(check, nodeId, settled);
  return {
    form,
    place: { nodeId, path },
    setNodeId,
    namer,
    check,
    verdict,
    usable: node !== undefined && settled === path && verdict?.usable === true,
  };
}

export type Editing = ReturnType<typeof usePlace>;
