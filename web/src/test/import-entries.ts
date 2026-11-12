import type {
  ImportsImportEntry,
  ImportsImportFile,
  WorksItem,
  WorksLibraryRow,
  WorksWorkView,
} from "../api/gen/model";
import { importingPack } from "./grabs";
import { cinema, libraryRow, orchard } from "./library";
import { homeId, seedboxId } from "./nodes";
import { filmsShelf, filmsVault, severanceId, shows, showsShelf } from "./pending-imports";

export const severanceRow: WorksLibraryRow = libraryRow({
  id: severanceId,
  key: "371980",
  label: "Severance",
  contentType: shows.id,
  rootId: showsShelf.id,
});

export const brazilRow: WorksLibraryRow = libraryRow({
  id: "0199f0c3-0000-7000-8000-000000001985",
  key: "68",
  label: "Brazil",
  rootId: filmsShelf.id,
});

export const vaultRow: WorksLibraryRow = libraryRow({
  id: "0199f0c3-0000-7000-8000-000000000949",
  key: "949",
  label: "Heat",
  rootId: filmsVault.id,
});

const episodeId = (season: number, episode: number) =>
  `0199f0d4-0000-7000-8000-0000000${String(season).padStart(2, "0")}${String(episode).padStart(3, "0")}`;

export const brazilItem: WorksItem = {
  id: "0199f0d4-0000-7000-8000-000000001985",
  levelId: null,
  key: "film",
  label: "Brazil",
  position: 1,
  withdrawnAt: null,
  monitored: true,
  aside: false,
  fields: [],
  ratings: [],
  artwork: [],
  availability: null,
};

const episode = (season: number, number: number, label: string) => ({
  ...brazilItem,
  id: episodeId(season, number),
  levelId: `season-${season}`,
  key: `${season}x${number}`,
  label,
  position: number,
  fields: [
    {
      key: "code",
      kind: "text" as const,
      text: `S${String(season).padStart(2, "0")}E${String(number).padStart(2, "0")}`,
    },
  ],
});

export const goodNews = episode(1, 1, "Good News About Hell");
export const halfLoop = episode(1, 2, "Half Loop");

export const severance: WorksWorkView = {
  ...orchard,
  id: severanceId,
  contentType: shows.id,
  key: "371980",
  label: "Severance",
  sortLabel: "Severance",
  fields: [],
  items: [goodNews, halfLoop],
};

export const brazilWork: WorksWorkView = {
  ...orchard,
  id: brazilRow.id,
  key: brazilRow.key,
  label: "Brazil",
  sortLabel: "Brazil",
  items: [brazilItem],
};

const gib = 1024 ** 3;

/** A file of a download or a folder Import, waiting or placed. */
const fileOf = (
  position: number,
  path: string,
  itemIds: string[],
  file: Partial<ImportsImportFile> = {},
): ImportsImportFile => ({
  position,
  path,
  size: 2 * gib,
  quality: "webdl-1080p",
  cut: "standard",
  itemIds,
  outcome: "waiting",
  reason: null,
  hold: null,
  fills: [],
  verdicts: [],
  recommended: null,
  actions: [],
  forces: [],
  placedBy: null,
  fileId: null,
  staged: 0,
  movable: false,
  ...file,
});

const importOf = (
  entry: Partial<ImportsImportEntry> & Pick<ImportsImportEntry, "id" | "source" | "files">,
): ImportsImportEntry => ({
  stage: "waiting",
  grab: null,
  nodeId: null,
  root: null,
  folder: "",
  workId: null,
  edition: null,
  moveFiles: false,
  seeding: false,
  failure: null,
  contentType: cinema.id,
  hint: { title: "", year: null, ids: [] },
  suggestion: null,
  createdAt: "2026-11-07T03:12:00Z",
  ignoredAt: null,
  finishedAt: null,
  ...entry,
});

const hd = "1f3e5a7c-9b2d-4f6e-8a0c-2e4f6a8c0b1d";

/**
 * Severance's season pack, which waits on episode 1: the Work's 1080p file
 * is at least as good. It can be forced in, deleting that file. Episode 2
 * placed by hard link.
 */
export const packEntry: ImportsImportEntry = importOf({
  id: importingPack.id,
  source: "grab",
  grab: { id: importingPack.id, release: importingPack.releaseTitle, stage: "importing" },
  nodeId: homeId,
  workId: severanceId,
  contentType: shows.id,
  seeding: true,
  hint: { title: "Severance", year: null, ids: [] },
  files: [
    fileOf(0, "Severance.S01E01.1080p.WEB-DL.mkv", [goodNews.id], {
      reason: {
        code: "not_an_upgrade",
        sentence: "This isn't better than the file you have for 1080p.",
      },
      verdicts: [{ targetId: hd, sentence: "This isn't better than the file you have for 1080p." }],
      recommended: "decline",
      actions: ["force", "notThisRelease"],
      forces: [
        {
          cell: { itemId: goodNews.id, edition: "any", targetId: hd },
          deletes: ["Severance/Season 1/Severance - S01E01 - Good News About Hell.mkv"],
        },
      ],
      staged: 2 * gib,
    }),
    fileOf(1, "Severance.S01E02.1080p.WEB-DL.mkv", [halfLoop.id], {
      outcome: "placed",
      fills: [{ itemId: halfLoop.id, edition: "any", targetId: hd }],
      placedBy: "link",
      fileId: "0199f0e5-0000-7000-8000-000000000102",
    }),
  ],
});

/** A download on seedbox whose one file waits on a name another file already has. */
export const takenEntry: ImportsImportEntry = importOf({
  id: "0199f0b2-0000-7000-8000-0000000000aa",
  source: "grab",
  grab: {
    id: "0199f0b2-0000-7000-8000-0000000000aa",
    release: "Brazil.1985.1080p.BluRay",
    stage: "importing",
  },
  nodeId: seedboxId,
  workId: brazilRow.id,
  seeding: true,
  hint: { title: "Brazil", year: 1985, ids: [] },
  files: [
    fileOf(0, "Brazil.1985.1080p.BluRay.mkv", [brazilItem.id], {
      quality: "bluray-1080p",
      reason: {
        code: "name_taken",
        sentence:
          "Another file already has the name Brazil (1985).mkv. Rename or move that file, then retry.",
      },
      recommended: "retry",
      actions: ["turnOnRenaming", "decline", "notThisRelease"],
    }),
  ],
});

const downloadsFolder = "/downloads/Films";

/** The folder an Admin picked, whose files match two Works: one Import per Work. */
export const folderEntries: ImportsImportEntry[] = [
  importOf({
    id: "0199f0b2-0000-7000-8000-0000000000f1",
    source: "folder",
    nodeId: homeId,
    folder: downloadsFolder,
    workId: brazilRow.id,
    hint: { title: "Brazil", year: 1985, ids: [] },
    files: [
      fileOf(0, `${downloadsFolder}/Brazil (1985).mkv`, [brazilItem.id], {
        reason: { code: "confirm", sentence: "Fills 1080p for Brazil (1985)." },
        fills: [{ itemId: brazilItem.id, edition: "any", targetId: hd }],
        recommended: "confirm",
        actions: ["match", "decline"],
        movable: true,
      }),
    ],
  }),
  importOf({
    id: "0199f0b2-0000-7000-8000-0000000000f2",
    source: "folder",
    nodeId: homeId,
    folder: downloadsFolder,
    workId: vaultRow.id,
    hint: { title: "Heat", year: 1995, ids: [] },
    files: [
      fileOf(0, `${downloadsFolder}/Heat (1995).mkv`, [], {
        reason: {
          code: "meets_nothing",
          sentence: "This doesn't meet any Target this Work wants. 1080p needs Bluray or WEB.",
        },
        recommended: "decline",
        actions: ["match"],
        movable: true,
      }),
    ],
  }),
];
