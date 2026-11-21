import { Link } from "@tanstack/react-router";

import type {
  DownloadclientsTorrentDetail,
  DownloadclientsTorrentGrabStage,
} from "../api/gen/model";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { type Clock, bytes, sinceWords } from "../format";
import { Data } from "../ui/data";
import { Empty } from "../ui/empty";
import { Fact, Facts } from "../ui/facts";
import { InspectorSection } from "../ui/inspector";
import type { Names } from "./names";

const grabStageWords: { readonly [S in DownloadclientsTorrentGrabStage]: string } = {
  downloading: "Downloading",
  importing: "Importing",
  waiting: "Waiting in Pending imports",
  imported: "Imported",
  failed: "Failed",
};

export function Grabs({ detail, names }: { detail: DownloadclientsTorrentDetail; names: Names }) {
  return (
    <InspectorSection title="Grabs">
      {detail.grabs.length === 0 ? (
        <Empty look="inline">No Grab uses it any more.</Empty>
      ) : (
        <ul>
          {detail.grabs.map((grab) => (
            <li
              key={grab.id}
              className="flex flex-col gap-0.5 py-2 not-first:shadow-[inset_0_1px_0_var(--color-hairline)]"
            >
              <span className="flex items-baseline justify-between gap-3">
                <Link
                  to="/works/$workId"
                  params={{ workId: grab.workId }}
                  className="truncate rounded-item font-semibold hover:underline"
                >
                  {names.work(grab.workId) ?? "A deleted Work"}
                </Link>
                <span className="shrink-0 text-ink-2">{grabStageWords[grab.stage]}</span>
              </span>
              <Data className="truncate text-meta text-ink-3">{grab.release}</Data>
            </li>
          ))}
        </ul>
      )}
    </InspectorSection>
  );
}

export function Machinery({
  detail,
  clock,
}: {
  detail: DownloadclientsTorrentDetail;
  clock: Clock;
}) {
  const types = useListContentTypesSuspense().data.contentTypes;
  return (
    <InspectorSection title="Details">
      <Facts look="inline">
        <Fact label="Info hash">
          <Data>{detail.infoHash}</Data>
        </Fact>
        <Fact label="Save folder">
          {detail.saveFolder === undefined ? "Not reported yet" : <Data>{detail.saveFolder}</Data>}
        </Fact>
        <Fact label="Added">{sinceWords(detail.addedAt, clock)}</Fact>
        <Fact label="Piece size">
          <Data>{bytes(detail.pieceSize)}</Data>
        </Fact>
        <Fact label="Private">
          {detail.private ? "Yes, peers come only from its trackers" : "No"}
        </Fact>
        <Fact label="Content Type">
          {types.find((type) => type.id === detail.contentType)?.name ?? detail.contentType}
        </Fact>
      </Facts>
    </InspectorSection>
  );
}
