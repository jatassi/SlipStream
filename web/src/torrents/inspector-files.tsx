import { useQueries } from "@tanstack/react-query";
import { useState } from "react";

import type { DownloadclientsTorrentDetail } from "../api/gen/model";
import { getGetWorkQueryKey, getWork } from "../api/gen/query/works/works";
import { bytes } from "../format";
import { Button } from "../ui/button";
import { Data } from "../ui/data";
import { InspectorSection } from "../ui/inspector";
import { percentOf } from "./model";

function useItemLabels(detail: DownloadclientsTorrentDetail): (id: string) => string | undefined {
  const workIds = [...new Set(detail.grabs.map((grab) => grab.workId))];
  const trees = useQueries({
    queries: workIds.map((workId) => ({
      queryKey: getGetWorkQueryKey(workId),
      queryFn: ({ signal }: { signal: AbortSignal }) => getWork(workId, { signal }),
    })),
  });
  const items = new Map(
    trees.flatMap((tree) => tree.data?.items.map((item) => [item.id, item.label] as const) ?? []),
  );
  return (id) => items.get(id);
}

const filesShown = 10;

export function Files({ detail }: { detail: DownloadclientsTorrentDetail }) {
  const item = useItemLabels(detail);
  const [all, setAll] = useState(false);
  const shown = all ? detail.files : detail.files.slice(0, filesShown);
  return (
    <InspectorSection title="Files">
      <ul>
        {shown.map((file) => {
          const label = file.itemId === undefined ? undefined : item(file.itemId);
          return (
            <li
              key={file.index}
              className="flex flex-col gap-0.5 py-2 not-first:shadow-[inset_0_1px_0_var(--color-hairline)]"
            >
              <span className="flex items-baseline justify-between gap-3">
                <Data className="min-w-0 wrap-anywhere">{file.path}</Data>
                {file.done !== undefined && (
                  <Data className="shrink-0 text-ink-2">
                    {percentOf({ done: file.done, wanted: file.size })}%
                  </Data>
                )}
              </span>
              <span className="text-meta text-ink-3">
                <Data>{bytes(file.size)}</Data>
                {label !== undefined && ` · Filled ${label}`}
              </span>
            </li>
          );
        })}
      </ul>
      {detail.files.length > filesShown && (
        <Button
          look="ghost"
          size="small"
          onClick={() => {
            setAll(!all);
          }}
        >
          {all ? "Show fewer" : `Show all ${detail.files.length} files`}
        </Button>
      )}
    </InspectorSection>
  );
}
