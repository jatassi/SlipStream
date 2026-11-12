import { useQuery } from "@tanstack/react-query";
import { Suspense } from "react";

import { sentenceOf } from "../api/errors";
import type { GrabsGrab } from "../api/gen/model";
import {
  actOnTorrents,
  useListTorrentsSuspense,
} from "../api/gen/query/downloadclients/downloadclients";
import { useProjection } from "../library/projection";
import { subjectOf } from "../torrents/actions";
import { Preview, freesWords } from "../torrents/remove-preview";
import { Spinner } from "../ui/spinner";

// A downloading Grab's cancel is its Torrent's remove, so it shows the same dry run.
function TorrentGoes({
  grab,
  torrentId,
  work,
}: {
  grab: GrabsGrab;
  torrentId: string;
  work: string;
}) {
  const works = useProjection().byId;
  const row = useListTorrentsSuspense(undefined, {
    query: { select: ({ rows }) => rows.find(({ id }) => id === torrentId) },
  }).data;
  const preview = useQuery({
    queryKey: ["cancel-preview", grab.id, torrentId],
    queryFn: ({ signal }) =>
      actOnTorrents({ action: "remove", ids: [torrentId], dryRun: true }, { signal }),
    gcTime: 0,
  });
  const subject = row === undefined ? { id: torrentId, name: work } : subjectOf(row, work);
  return (
    <>
      <p>
        {subject.keptIn === undefined
          ? `SlipStream removes its torrent and deletes the download copy${freesWords(preview.data?.preview?.frees)}. Files in your library stay.`
          : `SlipStream stops tracking its torrent and leaves it in ${subject.keptIn}. Files in your library stay.`}{" "}
        Its Item goes back to wanted.
      </p>
      {preview.isPending && <Spinner label="Checking what this changes" />}
      {preview.isError && <p className="text-fail-ink">{sentenceOf(preview.error)}</p>}
      {preview.data !== undefined && (
        <Preview
          parting="remove"
          acted={preview.data}
          subjects={[subject]}
          work={(id) => works.get(id)?.label}
          besides={grab.id}
        />
      )}
    </>
  );
}

export function Stops({ grab, work }: { grab: GrabsGrab; work: string }) {
  if (grab.stage === "downloading" && grab.torrentId !== null) {
    return (
      <Suspense fallback={<Spinner label="Checking what this changes" />}>
        <TorrentGoes grab={grab} torrentId={grab.torrentId} work={work} />
      </Suspense>
    );
  }
  if (grab.stage === "importing") {
    return (
      <p>
        SlipStream stops importing the download and keeps its torrent. Files already in your library
        stay, and Items it hasn&apos;t imported go back to wanted.
      </p>
    );
  }
  return <p>SlipStream stops fetching the Release. Its Item goes back to wanted.</p>;
}
