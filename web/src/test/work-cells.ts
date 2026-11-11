import type { FilesCellList, FilesFileList, FilesListedFile, GrabsGrab } from "../api/gen/model";
import { fileOf } from "../works/page/matrix-model";
import { downloading } from "./grabs";
import { orchardItemId } from "./library";
import { seedboxId } from "./nodes";
import { keeper, starter } from "./targets";

/** A download on seedbox, which waits while seedbox is away. */
export const awayGrab: GrabsGrab = { ...downloading, nodeId: seedboxId };

const hd = starter.id;
const uhd = keeper.id;
const base = { itemId: orchardItemId, edition: "any" } as const;

export const missing: FilesCellList = {
  cells: [
    {
      ...base,
      targetId: hd,
      targetName: "1080p",
      wanted: true,
      state: "missing",
    },
    {
      ...base,
      targetId: uhd,
      targetName: "2160p HDR",
      wanted: true,
      state: "missing",
    },
  ],
  items: [{ itemId: orchardItemId, status: "missing" }],
  summary: { status: "missing", cells: { missing: 2 }, items: 1, itemStatus: { missing: 1 } },
  needsYou: [],
};

export const filled: FilesCellList = {
  cells: [
    {
      ...base,
      targetId: hd,
      targetName: "1080p",
      wanted: false,
      state: "met",
      file: {
        id: "2a4c6e8f-0b1d-4e3f-9a5c-7e9b1d3f5a7c",
        path: "movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv",
        size: 734_003_200,
        quality: "webdl-1080p",
        probe: { width: 1920, height: 1080 },
      },
    },
    {
      ...base,
      targetId: uhd,
      targetName: "2160p HDR",
      wanted: false,
      state: "downloading",
      grabId: downloading.id,
      file: null,
    },
  ],
  items: [{ itemId: orchardItemId, status: "downloading" }],
  summary: {
    status: "downloading",
    cells: { downloading: 1, met: 1 },
    items: 1,
    itemStatus: { downloading: 1 },
  },
  needsYou: [],
};

const remux = {
  id: "2a4c6e8f-0b1d-4e3f-9a5c-7e9b1d3f5a7c",
  path: "movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 2160p.mkv",
  size: 734_003_200,
  quality: "remux-2160p",
  probe: { width: 3840, height: 2160 },
} as const;

export const filledTwice: FilesCellList = {
  ...filled,
  cells: [
    { ...base, targetId: uhd, targetName: "2160p HDR", wanted: false, state: "met", file: remux },
    { ...base, targetId: hd, targetName: "1080p", wanted: false, state: "met", file: remux },
  ],
  summary: { status: "met", cells: { met: 2 }, items: 1, itemStatus: { met: 1 } },
  items: [{ itemId: orchardItemId, status: "met" }],
};

export const hdtv = {
  id: "4c6e8a0b-2d4f-4a6c-8e0b-2d4f6a8c0e1f",
  path: "movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv",
  size: 524_288_000,
  quality: "hdtv-1080p",
  probe: { width: 1920, height: 1080 },
} as const;

export const upgradable: FilesCellList = {
  cells: [
    { ...base, targetId: hd, targetName: "1080p", wanted: true, state: "upgradable", file: hdtv },
    {
      ...base,
      targetId: uhd,
      targetName: "2160p HDR",
      wanted: true,
      state: "missing",
    },
  ],
  items: [{ itemId: orchardItemId, status: "missing" }],
  summary: {
    status: "missing",
    cells: { missing: 1, upgradable: 1 },
    items: 1,
    itemStatus: { missing: 1 },
  },
  needsYou: [],
};

/** 2160p HDR failed on the Grab, holding 1080p back with it. */
export const failedCells = (failed: string): FilesCellList => ({
  cells: [
    { ...base, targetId: hd, targetName: "1080p", wanted: true, state: "missing" },
    {
      ...base,
      targetId: uhd,
      targetName: "2160p HDR",
      wanted: true,
      state: "failed",
      grabId: failed,
      file: null,
    },
  ],
  items: [{ itemId: orchardItemId, status: "failed" }],
  summary: {
    status: "failed",
    cells: { failed: 1, missing: 1 },
    items: 1,
    itemStatus: { failed: 1 },
  },
  needsYou: [
    {
      kind: "failed",
      itemIds: [orchardItemId],
      edition: "any",
      targetId: uhd,
      waiting: [hd],
      grabId: failed,
    },
  ],
});

/** The library files the cells hold, each filling its own cell. */
export function filesOf({ cells }: FilesCellList): FilesFileList {
  const files = new Map<string, FilesListedFile>();
  for (const cell of cells) {
    const filling = fileOf(cell);
    if (filling !== null) {
      const fill = { itemId: cell.itemId, edition: cell.edition, targetId: cell.targetId };
      const known = files.get(filling.id);
      const { probe, ...file } = filling;
      files.set(filling.id, {
        ...file,
        folder: "movies/The Quiet Orchard (2021)",
        codec: "x264",
        cut: "standard",
        label: "",
        kept: false,
        fills: [...(known?.fills ?? []), fill],
        modifiedAt: null,
        goneAt: null,
        probeError: null,
        media: {
          container: "matroska,webm",
          durationSeconds: 0,
          bitrate: 0,
          video: {
            ...probe,
            codec: "h264",
            profile: "",
            bitDepth: 8,
            frameRate: 0,
            hdr: ["SDR"],
            dolbyVision: null,
          },
          audio: [],
          subtitles: [],
        },
      });
    }
  }
  return { folder: "movies/The Quiet Orchard (2021)", files: [...files.values()] };
}

const scannedFile = {
  folder: "movies/The Quiet Orchard (2021)",
  cut: "standard",
  label: "",
  modifiedAt: "2026-10-30T21:14:00Z",
  goneAt: null,
  probeError: null,
} as const;

/** What ffprobe read of a 2160p HDR10 and Dolby Vision 8 file. */
const uhdMedia: NonNullable<FilesListedFile["media"]> = {
  container: "matroska,webm",
  durationSeconds: 6720,
  bitrate: 52_000_000,
  video: {
    codec: "hevc",
    profile: "Main 10",
    width: 3840,
    height: 2160,
    bitDepth: 10,
    frameRate: 23.976,
    hdr: ["DV", "HDR10"],
    dolbyVision: { profile: 8, level: 6, compatibility: 1 },
  },
  audio: [
    {
      codec: "truehd",
      profile: "Dolby TrueHD + Dolby Atmos",
      atmos: true,
      channels: 8,
      layout: "7.1",
      language: "en",
      default: true,
      title: "",
    },
    {
      codec: "ac3",
      profile: "",
      atmos: false,
      channels: 6,
      layout: "5.1(side)",
      language: "fr",
      default: false,
      title: "",
    },
  ],
  subtitles: [
    { codec: "subrip", image: false, language: "en", forced: false, default: false },
    { codec: "hdmv_pgs_subtitle", image: true, language: "fr", forced: true, default: false },
  ],
};

/**
 * The Quiet Orchard's files after a library scan: a 2160p file probed in
 * full, a 1080p file probed before SlipStream kept tracks, and a 720p file
 * the scan found gone.
 */
export const scannedFiles: FilesFileList = {
  folder: "movies/The Quiet Orchard (2021)",
  files: [
    {
      ...scannedFile,
      id: "6e8a0c2e-4f6a-4c8e-9a0c-2e4f6a8c0e2a",
      path: "movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 2160p.mkv",
      size: 21_474_836_480,
      quality: "remux-2160p",
      codec: "x265",
      kept: false,
      fills: [{ itemId: orchardItemId, edition: "any", targetId: uhd }],
      media: uhdMedia,
    },
    {
      ...scannedFile,
      id: "8a0c2e4f-6a8c-4e0a-8c2e-4f6a8c0e2a4c",
      path: "movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 1080p.mkv",
      size: 734_003_200,
      quality: "webdl-1080p",
      codec: "x264",
      kept: false,
      fills: [{ itemId: orchardItemId, edition: "any", targetId: hd }],
      media: null,
    },
    {
      ...scannedFile,
      id: "0c2e4f6a-8c0e-4a2c-9e4f-6a8c0e2a4c6e",
      path: "movies/The Quiet Orchard (2021)/The Quiet Orchard (2021) - 720p.mkv",
      size: 524_288_000,
      quality: "hdtv-720p",
      codec: "x264",
      kept: false,
      fills: [],
      goneAt: "2026-11-03T08:00:00Z",
      media: null,
    },
  ],
};
