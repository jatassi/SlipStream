import type { FormatsExplained } from "./formatsExplained.ts";
import type { FormatsReadFacts } from "./formatsReadFacts.ts";
import type { FormatsTriedTarget } from "./formatsTriedTarget.ts";
import type { FormatsWorkRef } from "./formatsWorkRef.ts";

export interface FormatsParsed {
  /** The Items the name covers of the one Work it was judged against */
  covers: string[];
  facts: FormatsReadFacts;
  /** Every Custom Format, the draft included, by name */
  formats: FormatsExplained[];
  name: string;
  /** What the name reads as, the Works it matches and the Targets it would fill, or why it would fill none */
  sentence: string;
  /** With one Work, its Profile's Targets judged against its held files; with several or none, every Target with no held file */
  targets: FormatsTriedTarget[];
  /** The facts in words, joined with a middle dot: Quality, HDR, audio and channels, release group and size */
  words: string;
  /** The given Work when the name matches it, else every Work whose title and year it gives */
  works: FormatsWorkRef[];
}
