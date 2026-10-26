import type {
  WorksOutcome,
  WorksOutcomeProblem,
  WorksSkippedOutcomeReason,
} from "../../api/gen/model";
import { countWords } from "../../format";
import {
  type ActionKind,
  type AnswerOf,
  type Entry,
  type Named,
  type Subject,
  type WorkAction,
  entryOf,
} from "./actions";

/** What an action's answer says: one sentence for what it did and skipped, and one per failure. */
export type Words = { readonly done: string | undefined; readonly failed: readonly string[] };

const anyWork = { one: "Work", many: "Works" } as const;

const list = new Intl.ListFormat("en", { type: "conjunction" });

// Past this many, Works are counted rather than named.
const namedAtMost = 3;

/** A few Works by their labels, or more counted with their type's noun, or "Works" across types. */
export function namedOf(subjects: readonly Subject[]): Named {
  const [first] = subjects;
  if (subjects.length <= namedAtMost) {
    return { words: list.format(subjects.map(({ label }) => label)), one: subjects.length === 1 };
  }
  const oneType = subjects.every(({ type }) => type.id === first?.type.id);
  const noun = first !== undefined && oneType ? first.type.work : anyWork;
  return { words: countWords(subjects.length, noun), one: false };
}

const verb = ({ one }: Named, single: string, plural: string) => (one ? single : plural);

/** Why a Work with files keeps its library root. */
export const filesStay = (named: Named): string =>
  `${named.words} ${verb(named, "has files, so it stays in its library root", "have files, so they stay in their library roots")}. Moving a Work that has files isn't supported yet.`;

type Skipping = { readonly already: (named: Named) => string; readonly source: string };

const skipWords: {
  readonly [R in WorksSkippedOutcomeReason]: (named: Named, skipping: Skipping) => string;
} = {
  already: (named, { already }) => already(named),
  work_has_files: (named) => filesStay(named),
  work_gone_upstream: (named, { source }) =>
    `${source} no longer lists ${named.words}, so ${verb(named, "it wasn't", "they weren't")} refreshed.`,
  work_not_found: (named) => `${named.words} ${verb(named, "is", "are")} no longer in the library.`,
};

const because = ({ title, detail }: WorksOutcomeProblem) => detail ?? `${title}.`;

type Said = {
  readonly failed: boolean;
  /** Works alike share a key, and so one sentence. */
  readonly key: string;
  readonly say: (named: Named) => string;
};

function saidOf<K extends ActionKind>(
  outcome: WorksOutcome,
  subject: Subject,
  action: WorkAction<K>,
  answer: AnswerOf<K>,
  entry: Entry<K>,
): Said {
  switch (outcome.outcome) {
    case "done":
      return { failed: false, key: "done", say: (named) => entry.done(named, action, answer) };
    case "skipped": {
      const { reason } = outcome;
      const { source } = subject.type;
      const already = (named: Named) => entry.already(named, action);
      return {
        failed: false,
        key: reason === "work_gone_upstream" ? `${reason}\n${source}` : reason,
        say: (named) => skipWords[reason](named, { already, source }),
      };
    }
    case "failed": {
      const { problem } = outcome;
      return {
        failed: true,
        key: `${problem.code}\n${because(problem)}`,
        say: (named) => `Couldn't ${entry.attempt(action)} ${named.words}. ${because(problem)}`,
      };
    }
    default:
      return outcome satisfies never;
  }
}

type Group = { readonly said: Said; readonly subjects: Subject[] };

/**
 * Says how an action went, Work by Work: what it did, why it skipped any,
 * and why it failed on any. Works alike are counted together, and a
 * skipped Work's type names its metadata source.
 */
export function outcomeWords<K extends ActionKind>(
  action: WorkAction<K>,
  answer: AnswerOf<K>,
  subjects: readonly Subject[],
): Words {
  const entry = entryOf(action);
  const subjectOf = new Map(subjects.map((subject) => [subject.id, subject]));
  const groups = new Map<string, Group>();
  for (const outcome of answer.outcomes) {
    const subject = subjectOf.get(outcome.workId);
    if (subject === undefined) {
      throw new Error(`the answer names ${outcome.workId}, which the request didn't`);
    }
    const said = saidOf(outcome, subject, action, answer, entry);
    const known = groups.get(said.key) ?? { said, subjects: [] };
    known.subjects.push(subject);
    groups.set(said.key, known);
  }
  const doneFirst = (group: Group) => (group.said.key === "done" ? 0 : 1);
  const sentences = (failed: boolean) =>
    [...groups.values()]
      .filter(({ said }) => said.failed === failed)
      .toSorted((a, b) => doneFirst(a) - doneFirst(b))
      .map(({ said, subjects: alike }) => said.say(namedOf(alike)));
  const done = sentences(false);
  return { done: done.length === 0 ? undefined : done.join(" "), failed: sentences(true) };
}
