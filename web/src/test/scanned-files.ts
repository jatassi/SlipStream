import type { FilesFileList, FilesListedFile } from "../api/gen/model";
import { orchardItemId } from "./library";
import { keeper, starter } from "./targets";

const hd = starter.id;
const uhd = keeper.id;

const scannedFile = {
  folder: "movies/The Quiet Orchard (2021)",
  cut: "standard",
  label: "",
  modifiedAt: "2026-10-30T21:14:00Z",
  goneAt: null,
  probeError: null,
  placedBy: null,
  grab: null,
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
