import type { NamingTypeNaming, WorksContentType } from "../api/gen/model";
import { type NamingCard, cardOf, recommendedValues } from "./card";

const grouped = new Intl.NumberFormat("en");

/** A lead's button: a save of settings, or the type's rename plan. */
export type LeadAction =
  | {
      readonly kind: "save";
      readonly label: string;
      readonly values: Readonly<Record<string, unknown>>;
    }
  | { readonly kind: "review"; readonly label: string };

/** What a card leads with: one sentence, and the one action that does what it says. */
export type Lead = {
  readonly sentence: string;
  readonly action?: LeadAction;
};

type Reading = {
  readonly noun: string;
  readonly card: NamingCard;
  readonly naming: NamingTypeNaming;
  /** The type's files whose paths differ from what its templates make. */
  readonly unmatched: number;
};

const renamingOff = ({ noun, card }: Reading): Lead | undefined =>
  card.rename.value
    ? undefined
    : {
        sentence: `Turn on renaming so Jellyfin and Emby show the files of each ${noun} as one item.`,
        action: { kind: "save", label: "Turn on renaming", values: { [card.rename.key]: true } },
      };

const groupingLost = ({ noun, card, naming }: Reading): Lead | undefined =>
  naming.problems.length === 0
    ? undefined
    : {
        sentence: `Jellyfin and Emby may show the files of a ${noun} as separate items with these templates. Use the recommended templates.`,
        action: {
          kind: "save",
          label: "Use recommended",
          values: recommendedValues(card, naming.problems),
        },
      };

const unmatchedFiles = ({ noun, unmatched }: Reading): Lead | undefined =>
  unmatched === 0
    ? undefined
    : {
        sentence:
          unmatched === 1
            ? `1 ${noun} file doesn't match the templates.`
            : `${grouped.format(unmatched)} ${noun} files don't match the templates.`,
        action: { kind: "review", label: "Review renames" },
      };

const rules = [renamingOff, groupingLost, unmatchedFiles];

/** The first rule that applies, in the order the card checks them. */
export function recommend(
  type: WorksContentType,
  naming: NamingTypeNaming,
  unmatched: number,
): Lead {
  const reading = { noun: type.work.one.toLowerCase(), card: cardOf(naming), naming, unmatched };
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
