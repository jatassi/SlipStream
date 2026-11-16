import type { GrabsLostAt, GrabsLostAtStep } from "../api/gen/model";
import { bytes, countWords } from "../format";

/** Whom a Release lost to: the run's Grab, or the first Release a search could take when it grabbed none. */
export type Reference = "grabbed" | "first";

const referenceWords: Record<Reference, { subject: string; object: string }> = {
  grabbed: { subject: "The grabbed release", object: "the grabbed release" },
  first: { subject: "The first-ranked release", object: "the first-ranked release" },
};

const percent = (factor: string): string => `${Math.round(Number(factor) * 100)}%`;

const seeders = { one: "seeder", many: "seeders" };

const unneeded = { one: "file SlipStream doesn't need", many: "files SlipStream doesn't need" };

type Step = (
  values: { own: string; ref: string; at: string },
  reference: { subject: string; object: string },
) => string;

const steps: Record<GrabsLostAtStep, Step> = {
  want: ({ own, ref }, { subject }) =>
    `It's for ${own}. ${subject} is for ${ref}, which comes first in the Profile.`,
  fewer_releases: ({ own, ref }, { subject }) =>
    `It has ${own} of the missing files. ${subject} has ${ref}, so fewer downloads are needed.`,
  tier: ({ own, ref, at }, { subject }) =>
    `Its quality is ${own}. ${subject} is ${ref}, a higher tier${at}.`,
  revision: ({ own, ref, at }, { subject }) =>
    `Its revision is ${own}. ${subject} is ${ref}, a newer revision${at}.`,
  score: ({ own, ref, at }, { subject }) => `Its score${at} is ${own}. ${subject} scores ${ref}.`,
  chosen_by_another: (_, { object }) =>
    `It ties ${object} on every step through score. SlipStream already picked that one for another missing file, so it comes first.`,
  fewer_unwanted: ({ own, ref }, { subject }) =>
    `It includes ${countWords(Number(own), unneeded)}. ${subject} includes ${countWords(Number(ref), unneeded)}.`,
  priority: ({ own, ref }, { subject }) =>
    `Its Indexer's priority is ${own}. ${subject}'s is ${ref}, and a lower number comes first.`,
  freeleech: ({ own, ref }, { subject }) =>
    `Its download counts ${percent(own)} toward your ratio. ${subject}'s download counts ${percent(ref)}.`,
  seeders: ({ own, ref }, { subject }) =>
    `It has ${countWords(Number(own), seeders)}. ${subject} has ${countWords(Number(ref), seeders)}.`,
  size: ({ own, ref }, { subject }) =>
    `It's ${bytes(Number(own))}. ${subject} is ${bytes(Number(ref))}, and the larger one comes first.`,
  title: (_, { object }) =>
    `It ties ${object} at every other step. Its title comes later in alphabetical order.`,
};

/**
 * Where a Release lost to the reference, in two sentences: its own value at
 * the step, then the reference's. target is the name of the Target the
 * tier, revision and score steps compare at, absent at the other steps and
 * once that Target is deleted.
 */
export const lostAtSentence = (
  { step, own, ref }: GrabsLostAt,
  reference: Reference,
  target: string | undefined,
): string =>
  steps[step](
    { own, ref, at: target === undefined ? "" : ` at ${target}` },
    referenceWords[reference],
  );
