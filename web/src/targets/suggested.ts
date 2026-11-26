import { z } from "zod";

import type { TargetsQuality } from "../api/gen/model";
import { PlaybackSuggestion } from "../api/gen/zod/slipStream.schemas.zod";
import type { Rule, TargetDraft } from "./draft";
import { drop } from "./order";

export const newTargetSearch = z.object({
  from: z
    .object({
      server: z.string().max(200),
      player: z.string().max(200),
      suggestion: PlaybackSuggestion,
    })
    .optional()
    .catch(undefined),
});

type SuggestedFrom = NonNullable<z.infer<typeof newTargetSearch>["from"]>;

const resolutions = ["480p", "720p", "1080p", "2160p"] as const;

const notOrAny = (values: readonly string[]): Rule =>
  values.length === 0 ? { mode: "any" } : { mode: "not", values };

export function suggestedDraft(
  draft: TargetDraft,
  suggestion: PlaybackSuggestion,
  qualities: readonly TargetsQuality[],
): TargetDraft {
  const highest = resolutions.indexOf(suggestion.highest);
  const above = qualities.filter(
    (quality) => quality.resolution !== null && resolutions.indexOf(quality.resolution) > highest,
  );
  return {
    ...draft,
    name: suggestion.name,
    order: above.reduce((order, quality) => drop(order, quality.key), draft.order),
    cinema: false,
    filters: {
      ...draft.filters,
      video_codec: { mode: "only", values: suggestion.videoCodecs },
      hdr: notOrAny(suggestion.notHdr),
      audio_codec: notOrAny(suggestion.notAudioCodecs),
      channels: notOrAny(suggestion.notChannels),
    },
  };
}

export const suggestedLine = ({ server, player }: SuggestedFrom): string =>
  `The Playback report suggests this Target for ${player} on ${server}. Check its qualities and filters, then create it.`;
