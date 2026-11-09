import type { NamingTypeNaming, WorksContentType } from "../api/gen/model";
import { type NamingCard, cardOf, recommendedValues } from "./card";

/** What a card leads with: one sentence, and the one action that does what it says. */
export type Lead = {
  readonly sentence: string;
  readonly action?: { readonly label: string; readonly values: Readonly<Record<string, unknown>> };
};

type Reading = {
  readonly noun: string;
  readonly card: NamingCard;
  readonly naming: NamingTypeNaming;
};

const renamingOff = ({ noun, card }: Reading): Lead | undefined =>
  card.rename.value
    ? undefined
    : {
        sentence: `Turn on renaming so Jellyfin and Emby show the files of each ${noun} as one item.`,
        action: { label: "Turn on renaming", values: { [card.rename.key]: true } },
      };

const groupingLost = ({ noun, card, naming }: Reading): Lead | undefined =>
  naming.problems.length === 0
    ? undefined
    : {
        sentence: `Jellyfin and Emby may show the files of a ${noun} as separate items with these templates. Use the recommended templates.`,
        action: { label: "Use recommended", values: recommendedValues(card, naming.problems) },
      };

const rules = [renamingOff, groupingLost];

/** The first rule that applies, in the order the card checks them. */
export function recommend(type: WorksContentType, naming: NamingTypeNaming): Lead {
  const reading = { noun: type.work.one.toLowerCase(), card: cardOf(naming), naming };
  for (const rule of rules) {
    const lead = rule(reading);
    if (lead !== undefined) {
      return lead;
    }
  }
  return {
    sentence: `Plex, Jellyfin and Emby show the files of each ${reading.noun} as one item.`,
  };
}
