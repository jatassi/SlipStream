import type {
  FormatsReadFacts,
  FormatsWorkRef,
  NamingTestedName,
  WorksContentType,
} from "../api/gen/model";
import { reasonOf } from "../imports/words";

/** Where a tested name would go: nowhere when it matches no single Work, else as its Work's card names it. */
export type Outcome =
  | { readonly state: "unmatched"; readonly works: readonly FormatsWorkRef[] }
  | {
      readonly state: "refused";
      readonly work: FormatsWorkRef;
      readonly type: WorksContentType;
    }
  | {
      readonly state: "tested";
      readonly work: FormatsWorkRef;
      readonly type: WorksContentType;
      readonly tested: NamingTestedName;
    };

export type TestedFile = {
  readonly name: string;
  readonly facts: FormatsReadFacts;
  readonly outcome: Outcome;
};

const list = new Intl.ListFormat("en", { type: "conjunction" });

const pending = "A library scan would send it to Pending imports.";

/** What a tested name matches and what happens to it, the path aside. */
export function leadOf(outcome: Outcome): string {
  switch (outcome.state) {
    case "unmatched":
      return outcome.works.length === 0
        ? `It matches no Work. ${pending}`
        : `It matches ${String(outcome.works.length)} Works: ${list.format(outcome.works.map(({ label }) => label))}. ${pending}`;
    case "refused":
      return `It matches ${outcome.work.label}. The ${outcome.type.name} card has a setting the save check refuses. Fix it to see where this file would go.`;
    case "tested": {
      const { tested, type, work } = outcome;
      if (tested.unplaced !== undefined) {
        return `It matches ${work.label}. ${reasonOf(tested.unplaced, type)}`;
      }
      if (tested.detail !== undefined) {
        return `It matches ${work.label}. ${tested.detail}`;
      }
      return `It matches ${work.label} and would be imported as:`;
    }
    default:
      return outcome satisfies never;
  }
}

/** The path in its root a tested name would be imported at, when it would be. */
export const pathOf = (outcome: Outcome): string | undefined =>
  outcome.state === "tested" && outcome.tested.path !== "" ? outcome.tested.path : undefined;
