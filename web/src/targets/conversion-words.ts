import { z } from "zod";

import { isProblem } from "../api/errors";
import type {
  TargetsConversionSpecAudio,
  TargetsConversionSpecCodec,
  TargetsConversionSpecContainer,
  TargetsConversionSpecCurve,
  TargetsConversionSpecRange,
  TargetsProblem,
  TargetsProblemCode,
} from "../api/gen/model";
import type { TargetDraft } from "./draft";

type Option<V extends string> = { readonly value: V; readonly label: string };

export const codecs: readonly Option<TargetsConversionSpecCodec>[] = [
  { value: "h264", label: "H.264" },
  { value: "hevc", label: "HEVC" },
  { value: "av1", label: "AV1" },
];

export const heights: readonly Option<string>[] = [
  { value: "2160", label: "2160p" },
  { value: "1080", label: "1080p" },
  { value: "720", label: "720p" },
  { value: "480", label: "480p" },
];

export const depths: readonly Option<string>[] = [
  { value: "8", label: "8 bit" },
  { value: "10", label: "10 bit" },
];

export const ranges: readonly Option<TargetsConversionSpecRange>[] = [
  { value: "sdr", label: "SDR" },
  { value: "hdr10", label: "HDR10" },
  { value: "source", label: "Same as the source" },
];

export const audios: readonly Option<TargetsConversionSpecAudio>[] = [
  { value: "copy", label: "Copy the original tracks" },
  { value: "stereo_aac", label: "One stereo AAC track" },
  { value: "ac3_51_and_stereo_aac", label: "A 5.1 AC-3 track and a stereo AAC track" },
];

const audioMade: readonly Option<TargetsConversionSpecAudio>[] = [
  { value: "stereo_aac", label: "stereo AAC" },
  { value: "ac3_51_and_stereo_aac", label: "5.1 AC-3 and stereo AAC" },
];

export const containers: readonly Option<TargetsConversionSpecContainer>[] = [
  { value: "mkv", label: "MKV" },
  { value: "mp4", label: "MP4" },
];

export const curves: readonly Option<TargetsConversionSpecCurve>[] = [
  { value: "bt2390", label: "BT.2390" },
  { value: "hable", label: "Hable" },
  { value: "mobius", label: "Möbius" },
  { value: "reinhard", label: "Reinhard" },
];

const grouped = new Intl.NumberFormat("en");

const labelOf = (options: readonly Option<string>[], value: string) =>
  options.find((option) => option.value === value)?.label ?? value;

const kbpsWords = (kbps: string) => `${grouped.format(Number(kbps))} kbps`;

export const olderWords = (older: number) =>
  older === 1
    ? "1 file was made with older settings."
    : `${grouped.format(older)} files were made with older settings.`;

type Words = (facts: Readonly<Record<string, string>>, draft: TargetDraft) => string;

const sizeWords: Words = (facts, draft) => {
  const low = kbpsWords(facts["minKbps"] ?? "");
  const high = facts["maxKbps"];
  const range = high === undefined ? `at least ${low}` : `${low} to ${kbpsWords(high)}`;
  return `At ${kbpsWords(facts["kbps"] ?? "")}, files fall outside this Target's ${String(draft.conversion?.maxHeight)}p sizes, which take ${range}. Change the bitrate or the sizes.`;
};

const filteredWords: Words = (facts) =>
  facts["audio"] === undefined
    ? `These settings make ${labelOf(codecs, facts["codec"] ?? "")} files, and this Target's video codec filter refuses them. Change the codec or the Target's filters.`
    : `These settings make ${labelOf(audioMade, facts["audio"])} audio, and this Target's audio filters refuse it. Change the audio or the Target's filters.`;

const wordsByCode: Readonly<Record<TargetsProblemCode, Words>> = {
  codec_filtered: filteredWords,
  quality_unranked: ({ height = "" }) =>
    `This Target accepts no ${height}p quality, so it refuses every file these settings make. Change the largest resolution, or accept a ${height}p quality.`,
  outside_size: sizeWords,
  range_from_sdr: () =>
    "An SDR source always makes an SDR file, and this Target's HDR filter refuses SDR. Let the HDR filter allow SDR.",
  container_audio: () =>
    "MP4 can't hold TrueHD or DTS audio. Choose stereo AAC or 5.1 AC-3, or choose MKV.",
};

/** A conversion_refused problem in words. Which settings refuse is the server's to decide. */
export const problemWords = ({ code, facts }: TargetsProblem, draft: TargetDraft): string =>
  wordsByCode[code](facts, draft);

const Refused = z.object({
  problems: z.array(
    z.object({
      code: z.enum([
        "codec_filtered",
        "range_from_sdr",
        "outside_size",
        "container_audio",
        "quality_unranked",
      ]),
      field: z.string(),
      facts: z.record(z.string(), z.string()),
    }),
  ),
});

/** A save's refusal of the Conversion settings, worded; any other error as it came. */
export function refusedConversion(error: unknown, draft: TargetDraft): unknown {
  if (!isProblem(error, "conversion_refused")) {
    return error;
  }
  const { problems } = Refused.parse(error.body);
  return new Error(problems.map((problem) => problemWords(problem, draft)).join(" "));
}
