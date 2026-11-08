import type {
  FilesAudioTrack,
  FilesHdrFormat,
  FilesListedFile,
  FilesVideoInfo,
} from "../../api/gen/model";
import { isProblem, sentenceOf } from "../../api/errors";
import { useForgetFiles } from "../../api/gen/query/files/files";
import { lengthWords } from "../../format";
import { Button } from "../../ui/button";
import { Data } from "../../ui/data";
import { Details } from "../../ui/details";
import { toast } from "../../ui/toast";
import { nameOf } from "../paths";

const languageNames = new Intl.DisplayNames(["en"], { type: "language" });

const languageOf = (code: string | null) =>
  code === null ? "Unknown language" : (languageNames.of(code) ?? code);

const hdrWords: {
  readonly [H in FilesHdrFormat]: (video: FilesVideoInfo) => string;
} = {
  DV: ({ dolbyVision }) =>
    dolbyVision === null ? "Dolby Vision" : `Dolby Vision profile ${dolbyVision.profile}`,
  "HDR10+": () => "HDR10+",
  HDR10: () => "HDR10",
  HLG: () => "HLG",
  SDR: () => "SDR",
};

// The audio words a Target filters on, as media.Record.Probed reads them.
const audioCodecs: { readonly [codec: string]: string } = {
  truehd: "TrueHD",
  eac3: "DD+",
  ac3: "DD",
  aac: "AAC",
  flac: "FLAC",
  opus: "Opus",
  mp3: "MP3",
};

function audioCodecOf({ codec, profile }: FilesAudioTrack): string {
  if (codec === "dts") {
    return profile.startsWith("DTS-HD MA") ? "DTS-HD MA" : "DTS";
  }
  return codec.startsWith("pcm_") ? "PCM" : (audioCodecs[codec] ?? codec);
}

const layouts: { readonly [channels: number]: string } = { 8: "7.1", 6: "5.1", 2: "2.0", 1: "1.0" };

const channelsOf = ({ channels, layout }: FilesAudioTrack) =>
  layouts[channels] ?? (layout === "" ? `${channels} channels` : layout);

const audioWords = (track: FilesAudioTrack) =>
  [
    `${audioCodecOf(track)}${track.atmos ? " Atmos" : ""}`,
    channelsOf(track),
    languageOf(track.language),
    ...(track.title === "" ? [] : [track.title]),
  ].join(" · ");

function videoWords(codec: string, video: FilesVideoInfo): string {
  return [
    codec,
    `${video.width}×${video.height}`,
    ...(video.bitDepth > 0 ? [`${video.bitDepth}-bit`] : []),
    video.hdr.map((hdr) => hdrWords[hdr](video)).join(", "),
  ].join(" · ");
}

type MediaRow = { readonly label: string; readonly values: readonly string[] };

/** What ffprobe read of a file, a row per kind of stream; none for a file read before SlipStream kept its tracks. */
export function mediaRows({ media, codec, probeError }: FilesListedFile): readonly MediaRow[] {
  const failed = probeError === null ? [] : [{ label: "Read error", values: [probeError] }];
  if (media === null) {
    return failed;
  }
  const subtitles = [
    ...new Set(
      media.subtitles.map(
        ({ language, forced }) => `${languageOf(language)}${forced ? " (forced)" : ""}`,
      ),
    ),
  ];
  return [
    { label: "Video", values: [videoWords(codec ?? media.video.codec, media.video)] },
    { label: "Audio", values: media.audio.map(audioWords) },
    { label: "Subtitles", values: subtitles.length === 0 ? [] : [subtitles.join(", ")] },
    {
      label: "Duration",
      values: media.durationSeconds > 0 ? [lengthWords(media.durationSeconds * 1000)] : [],
    },
    ...failed,
  ].filter(({ values }) => values.length > 0);
}

/** A file's media info under Details, or nothing when there is none. */
export function FileMedia({ file }: { file: FilesListedFile }) {
  const rows = mediaRows(file);
  if (rows.length === 0) {
    return null;
  }
  return (
    <Details>
      <dl className="flex flex-col gap-2 pb-1">
        {rows.map(({ label, values }) => (
          <div key={label}>
            <dt className="text-label text-ink-3">{label}</dt>
            {values.map((value, index) => (
              // oxlint-disable-next-line react/no-array-index-key -- two tracks can read the same, and the order never changes
              <dd key={index} className="break-words whitespace-normal text-ink-1">
                <Data>{value}</Data>
              </dd>
            ))}
          </div>
        ))}
      </dl>
    </Details>
  );
}

const day = new Intl.DateTimeFormat(undefined, { day: "numeric", month: "short" });

export const goneWords = (goneAt: string): string => `Gone since ${day.format(new Date(goneAt))}`;

/** Forgets a gone file. One already forgotten, as by another tab, counts as done. */
export function ForgetFile({ file }: { file: FilesListedFile }) {
  const forget = useForgetFiles({
    mutation: {
      onError: (error) => {
        if (!isProblem(error, "file_not_found")) {
          toast(sentenceOf(error), { kind: "fail" });
        }
      },
    },
  });
  return (
    <Button
      size="small"
      busy={forget.isPending}
      aria-label={`Forget ${nameOf(file.path)}`}
      onClick={() => {
        forget.mutate({ data: { fileIds: [file.id] } });
      }}
    >
      Forget
    </Button>
  );
}
