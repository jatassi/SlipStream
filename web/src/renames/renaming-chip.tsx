import { useQuery } from "@tanstack/react-query";

import type { CountMovingParams } from "../api/gen/model";
import { countMoving, getCountMovingQueryKey } from "../api/gen/query/files/files";
import { LiveChip } from "../ui/live-chip";
import { renamingWords } from "./renames-words";

/** "Renaming 128 files…" while the disk queue still moves files of a Work, or of the library when left out. */
export function RenamingChip({ params }: { params?: CountMovingParams }) {
  const moving = useQuery({
    queryKey: getCountMovingQueryKey(params),
    queryFn: ({ signal }) => countMoving(params, { signal }),
  }).data?.files;
  return moving === undefined || moving === 0 ? null : <LiveChip>{renamingWords(moving)}</LiveChip>;
}
