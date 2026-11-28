import type {
  TargetsConversionSpec,
  TargetsConversionSpecDepth,
  TargetsConversionSpecMaxHeight,
  TargetsConversionView,
} from "../api/gen/model";

/** Conversion settings as typed; the bitrate is text until it's sent. */
export type ConversionDraft = Omit<TargetsConversionSpec, "kbps"> & { readonly kbps: string };

const leastKbps = 100;
const mostKbps = 200_000;

export const newConversion = (kbps: number | undefined): ConversionDraft => ({
  codec: "h264",
  maxHeight: 1080,
  kbps: kbps === undefined ? "" : String(kbps),
  depth: 8,
  range: "sdr",
  keepDolbyVision: false,
  curve: "bt2390",
  audio: "copy",
  dropCommentary: true,
  container: "mkv",
  runOnNodeId: null,
});

export const conversionDraftOf = (view: TargetsConversionView): ConversionDraft => ({
  codec: view.codec,
  maxHeight: view.maxHeight,
  kbps: String(view.kbps),
  depth: view.depth,
  range: view.range,
  keepDolbyVision: view.keepDolbyVision,
  curve: view.curve,
  audio: view.audio,
  dropCommentary: view.dropCommentary,
  container: view.container,
  runOnNodeId: view.runOnNodeId,
});

export const keepsDolbyVision = ({ codec, range }: Pick<ConversionDraft, "codec" | "range">) =>
  codec !== "h264" && range !== "sdr";

/** A change to the settings, with H.264 held to 8 bit and Dolby Vision kept only where it can be. */
export function changedConversion(
  conversion: ConversionDraft,
  change: Partial<ConversionDraft>,
): ConversionDraft {
  const next = { ...conversion, ...change };
  return {
    ...next,
    depth: next.codec === "h264" ? 8 : next.depth,
    keepDolbyVision: next.keepDolbyVision && keepsDolbyVision(next),
  };
}

export function kbpsProblem(kbps: string): string | undefined {
  const value = Number(kbps.trim());
  return Number.isInteger(value) && value >= leastKbps && value <= mostKbps
    ? undefined
    : "Enter a whole number of kbps from 100 to 200,000.";
}

const heights = [2160, 1080, 720, 480] as const satisfies readonly TargetsConversionSpecMaxHeight[];
const depths = [8, 10] as const satisfies readonly TargetsConversionSpecDepth[];

const parsed = <V extends number>(options: readonly V[], text: string): V => {
  const value = options.find((option) => String(option) === text);
  if (value === undefined) {
    throw new Error(`${text} isn't one of ${options.join(", ")}`);
  }
  return value;
};

export const heightOf = (text: string) => parsed(heights, text);
export const depthOf = (text: string) => parsed(depths, text);

export const conversionBody = ({ kbps, ...rest }: ConversionDraft): TargetsConversionSpec => ({
  ...rest,
  kbps: Number(kbps),
});
