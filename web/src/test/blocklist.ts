import type { GrabsBlockEntry, GrabsBlocklist } from "../api/gen/model";
import { longShelf } from "./anthology";
import { orchard, orchardId } from "./library";

export const blockedHash = "a1b2c3d4e5f60718293a4b5c6d7e8f9012345678";

const entry = (
  part: Partial<GrabsBlockEntry> & Pick<GrabsBlockEntry, "infoHash">,
): GrabsBlockEntry => ({
  title: "The.Quiet.Orchard.2021.1080p.BluRay.x264-SPARKS",
  words: "Bluray-1080p · SPARKS · 9.8 GB",
  cause: "unregistered",
  because: "TorrentLeech reported it unregistered.",
  indexer: "TorrentLeech",
  work: { id: orchardId, title: orchard.label },
  addedAt: "2026-11-02T08:30:00Z",
  ...part,
});

/** A Blocklist page: a release fault, Mark failed, a cancel, and one whose Work was deleted. */
export const blocklistPage: GrabsBlocklist = {
  entries: [
    entry({ infoHash: blockedHash }),
    entry({
      infoHash: "b2c3d4e5f60718293a4b5c6d7e8f901234567890",
      title: "The.Long.Shelf.V01S04.1080p.WEB-DL-NTb",
      words: "WEBDL-1080p · NTb · 1.2 GB",
      cause: "marked_failed",
      because: "Ada Okafor marked its download failed.",
      indexer: "Prowlarr",
      work: { id: longShelf.id, title: longShelf.label },
      addedAt: "2026-11-03T10:00:00Z",
    }),
    entry({
      infoHash: "c3d4e5f60718293a4b5c6d7e8f9012345678901a",
      title: "The.Quiet.Orchard.2021.2160p.WEB-DL.HDR-FLUX",
      words: "WEBDL-2160p · HDR · FLUX · 18 GB",
      cause: "by_hand",
      because: "Ada Okafor blocklisted it when canceling its Grab.",
      addedAt: "2026-11-04T11:15:00Z",
    }),
    entry({
      infoHash: "d4e5f60718293a4b5c6d7e8f9012345678901a2b",
      title: "Gone.Work.2019.1080p.WEB-DL-NTb",
      words: "WEBDL-1080p · NTb · 3.9 GB",
      cause: "no_payload",
      because: "It never downloaded any data.",
      indexer: null,
      work: null,
      addedAt: "2026-11-05T12:00:00Z",
    }),
  ],
  total: 4,
};
