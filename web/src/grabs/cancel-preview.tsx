import { useQuery } from "@tanstack/react-query";

import { sentenceOf } from "../api/errors";
import { actOnTorrents } from "../api/gen/query/downloadclients/downloadclients";
import { subjectOf } from "../torrents/actions";
import { Preview, freesWords } from "../torrents/remove-preview";
import { SignalLine } from "../ui/signal-line";
import { Spinner } from "../ui/spinner";
import type { Job } from "./job";

// A downloading Grab's cancel is its Torrent's remove, so it shows the same dry run.
function TorrentGoes({ grab, torrentId, work, torrent, names }: Job & { torrentId: string }) {
  const preview = useQuery({
    queryKey: ["cancel-preview", grab.id, torrentId],
    queryFn: ({ signal }) =>
      actOnTorrents({ action: "remove", ids: [torrentId], dryRun: true }, { signal }),
    gcTime: 0,
  });
  const subject = torrent === undefined ? { id: torrentId, name: work } : subjectOf(torrent, work);
  return (
    <>
      <p>
        {subject.keptIn === undefined
          ? `SlipStream removes its torrent and deletes the download copy${freesWords(preview.data?.preview?.frees)}. Files in your library stay.`
          : `SlipStream stops tracking its torrent and leaves it in ${subject.keptIn}. Files in your library stay.`}{" "}
        SlipStream counts the files it was for as missing again.
      </p>
      {preview.isPending && <Spinner label="Checking what this changes" />}
      {preview.isError && (
        <SignalLine tone="fail" announce>
          {sentenceOf(preview.error)}
        </SignalLine>
      )}
      {preview.data !== undefined && (
        <Preview
          parting="remove"
          acted={preview.data}
          subjects={[subject]}
          work={names}
          besides={grab.id}
        />
      )}
    </>
  );
}

export function Stops(job: Job) {
  const { grab } = job;
  if (grab.stage === "downloading" && grab.torrentId !== null) {
    return <TorrentGoes {...job} torrentId={grab.torrentId} />;
  }
  if (grab.stage === "importing") {
    return (
      <p>
        SlipStream stops importing the download and keeps its torrent. Files already in your library
        stay, and files it hasn&apos;t imported count as missing again.
      </p>
    );
  }
  return (
    <p>SlipStream stops fetching the release and counts the files it was for as missing again.</p>
  );
}
