import { useInfiniteQuery } from "@tanstack/react-query";

import type { ReconcilingGroup } from "../api/gen/model";
import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import { Button } from "../ui/button";
import { List } from "../ui/list";
import { Section } from "../ui/section";
import { GroupAction, TorrentRow } from "./review-actions";
import { groupQuery } from "./review-query";
import { groupWords } from "./words";

const titles = {
  files: "Files found",
  no_files: "No files",
  not_looked: "Not looked for yet",
} as const;

export function GroupSection({ group }: { group: ReconcilingGroup }) {
  const { nodes } = useListNodesSuspense().data;
  const name = (id: string) => nodes.find((node) => node.id === id)?.name ?? "a removed Node";
  const { data, hasNextPage, fetchNextPage, isFetchingNextPage } = useInfiniteQuery(
    groupQuery(group),
  );
  const rows = data?.pages.flatMap((page) => page.torrents) ?? [];
  const title = group.reason === "conflict" ? "On two Nodes" : titles[group.found];
  return (
    <Section title={title} count={group.count} action={<GroupAction group={group} />}>
      <div className="flex flex-col gap-3">
        <p className="max-w-[64ch] text-ink-2">{groupWords(group).sentence}</p>
        <List label={title}>
          {rows.map((torrent) => (
            <TorrentRow key={torrent.id} torrent={torrent} node={name(torrent.nodeId)} />
          ))}
        </List>
        {hasNextPage && (
          <div className="flex justify-center">
            <Button busy={isFetchingNextPage} onClick={() => void fetchNextPage()}>
              Load more
            </Button>
          </div>
        )}
      </div>
    </Section>
  );
}
