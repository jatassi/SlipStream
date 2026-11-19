import type { ReactNode } from "react";

import type { TargetsQualityKey, TargetsQualitySource } from "../api/gen/model";
import { useListQualitiesSuspense } from "../api/gen/query/targets/targets";
import { Chip } from "../ui/chip";
import { type Definition, Defined, type Term } from "../ui/defined";

/** What each source means, which a quality's name opens. A source the wire adds fails the build until it has one. */
const sourceDefinitions = {
  tv: "The video is recorded from a standard-definition TV broadcast.",
  hdtv: "The video is recorded from a high-definition TV broadcast.",
  dvd: "The video is encoded from a DVD.",
  webrip:
    "The video is captured from a streaming service as it plays and encoded again, so it can lose some detail.",
  webdl: "The video is downloaded from a streaming service exactly as the service sends it.",
  bluray: "The video is encoded from a Blu-ray disc into a smaller file.",
  remux:
    "The video is copied from a Blu-ray disc without encoding it again, so it keeps all of the disc's detail.",
  cinema: "The video is recorded in a cinema, often with a handheld camera.",
} as const satisfies { readonly [S in TargetsQualitySource]: Definition };

const looks = {
  chip: (word: ReactNode) => <Chip look="outline">{word}</Chip>,
  word: (word: ReactNode) => word,
} as const;

/** The quality's name and its source's definition, from GET /qualities, which every key the wire sends is in. */
function useQualityTerm(key: TargetsQualityKey): Term {
  const { qualities } = useListQualitiesSuspense().data;
  const quality = qualities.find((each) => each.key === key);
  if (quality === undefined) {
    throw new Error(`GET /qualities has no quality ${key}`);
  }
  return { word: quality.name, definition: sourceDefinitions[quality.source] };
}

/**
 * A quality by its name, such as WEBDL-1080p, which opens its source's
 * definition on hover, tap and Enter. chip is the outline chip, for a quality
 * on its own, and word sits inside a line of text.
 */
export function Quality({
  quality,
  look = "chip",
}: {
  quality: TargetsQualityKey;
  look?: keyof typeof looks;
}) {
  return <Defined term={useQualityTerm(quality)}>{looks[look]}</Defined>;
}

/** A quality's name alone, for a quality inside another control, such as a row's button, where a definition can't open. */
export function QualityName({ quality }: { quality: TargetsQualityKey }) {
  return useQualityTerm(quality).word;
}
