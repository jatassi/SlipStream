import type { TargetsFilterAttribute } from "../api/gen/model";

export type Attribute = TargetsFilterAttribute;

export type Value = { readonly key: string; readonly name: string };

const plain = (keys: readonly string[]): Value[] => keys.map((key) => ({ key, name: key }));

const languageNames = new Intl.DisplayNames(["en"], { type: "language" });

// The languages a picker offers. Any ISO 639 code is valid on the wire;
// these are the ones people pick most.
const languageCodes = [
  "en",
  "fr",
  "de",
  "es",
  "it",
  "pt",
  "nl",
  "sv",
  "no",
  "da",
  "fi",
  "pl",
  "cs",
  "hu",
  "ro",
  "el",
  "tr",
  "ru",
  "uk",
  "he",
  "ar",
  "fa",
  "hi",
  "ta",
  "te",
  "th",
  "vi",
  "id",
  "ja",
  "ko",
  "zh",
] as const;

export const attributes: readonly {
  readonly key: Attribute;
  readonly label: string;
  readonly values: readonly Value[];
}[] = [
  { key: "hdr", label: "HDR", values: plain(["DV", "HDR10+", "HDR10", "HLG", "SDR"]) },
  {
    key: "video_codec",
    label: "Video codec",
    values: plain(["x264", "x265", "AV1", "VP9", "XviD", "MPEG2"]),
  },
  {
    key: "audio_codec",
    label: "Audio codec",
    values: plain(["TrueHD", "DTS-HD MA", "DTS", "DD+", "DD", "AAC", "FLAC", "PCM", "Opus", "MP3"]),
  },
  { key: "channels", label: "Channels", values: plain(["7.1", "5.1", "2.0", "1.0"]) },
  {
    key: "language",
    label: "Audio language",
    values: [
      { key: "original", name: "Original language" },
      ...languageCodes.map((key) => ({ key, name: languageNames.of(key) ?? key })),
    ],
  },
];
