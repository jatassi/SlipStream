import { Link } from "@tanstack/react-router";

import type {
  DownloadclientsTorrentDetail,
  DownloadclientsTorrentGrabStage,
} from "../api/gen/model";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { type Clock, bytes, sinceWords } from "../format";
import { Data } from "../ui/data";
import { InspectorSection } from "../ui/inspector";
import { FactList } from "./inspector-facts";
import type { Names } from "./names";

const grabStageWords: { readonly [S in DownloadclientsTorrentGrabStage]: string } = {
  downloading: "Downloading",
  importing: "Importing",
  imported: "Imported",
  failed: "Failed",
};

export function Grabs({ detail, names }: { detail: DownloadclientsTorrentDetail; names: Names }) {
  return (
    <InspectorSection title="Grabs">
      {detail.grabs.length === 0 ? (
        <p className="text-ink-3">No Grab uses it any more.</p>
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
      <FactList
        facts={[
          ["Info hash", <Data key="h">{detail.infoHash}</Data>],
          [
            "Save folder",
            detail.saveFolder === undefined ? (
              "Not reported yet"
            ) : (
              <Data key="f">{detail.saveFolder}</Data>
            ),
          ],
          ["Added", sinceWords(detail.addedAt, clock)],
          ["Piece size", <Data key="p">{bytes(detail.pieceSize)}</Data>],
          ["Private", detail.private ? "Yes, peers come only from its announce URLs" : "No"],
          [
            "Content Type",
            types.find((type) => type.id === detail.contentType)?.name ?? detail.contentType,
          ],
        ]}
      />
    </InspectorSection>
  );
}
