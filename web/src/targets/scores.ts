import type { FormatsCustomFormat, TargetsScores } from "../api/gen/model";

/** One Custom Format's score at a Target, as typed. */
export type FormatScore = { readonly id: string; readonly name: string; readonly score: string };

/** A Custom Format as the Scores row names it. */
export type FormatName = Pick<FormatsCustomFormat, "id" | "name">;

/** A Target's scores as typed: every Custom Format by name, and the three totals. */
export type ScoresDraft = {
  readonly formats: readonly FormatScore[];
  readonly minimum: string;
  readonly upgradeUntil: string;
  readonly increment: string;
};

export type Totals = Pick<TargetsScores, "minimum" | "upgradeUntil" | "increment">;

const whole = /^-?\d+$/u;

export const scoreProblem = (text: string): string | undefined =>
  whole.test(text.trim()) ? undefined : "Enter a whole number.";

export const incrementProblem = (text: string): string | undefined =>
  whole.test(text.trim()) && Number(text) >= 1 ? undefined : "Enter a whole number of 1 or more.";

export function scoresDraftOf(scores: TargetsScores, formats: readonly FormatName[]): ScoresDraft {
  const scored = new Map(scores.formats.map(({ formatId, score }) => [formatId, score]));
  return {
    formats: formats.map(({ id, name }) => ({ id, name, score: String(scored.get(id) ?? 0) })),
    minimum: String(scores.minimum),
    upgradeUntil: String(scores.upgradeUntil),
    increment: String(scores.increment),
  };
}

export const withScore = (scores: ScoresDraft, id: string, score: string): ScoresDraft => ({
  ...scores,
  formats: scores.formats.map((format) => (format.id === id ? { ...format, score } : format)),
});

export const scoresProblem = (scores: ScoresDraft): boolean =>
  [scores.minimum, scores.upgradeUntil, ...scores.formats.map(({ score }) => score)].some(
    (text) => scoreProblem(text) !== undefined,
  ) || incrementProblem(scores.increment) !== undefined;

// The sentences follow the draft while a box is mid-edit, reading it as its
// default; problemsOf stops a save until it is a whole number.
const read = (text: string, problem: (text: string) => string | undefined, otherwise: number) =>
  problem(text) === undefined ? Number(text) : otherwise;

export const totalsOf = (scores: ScoresDraft): Totals => ({
  minimum: read(scores.minimum, scoreProblem, 0),
  upgradeUntil: read(scores.upgradeUntil, scoreProblem, 0),
  increment: read(scores.increment, incrementProblem, 1),
});

/** The body's scores, once problemsOf finds nothing. A format at 0 is left out, as the API reads it. */
export const scoresBody = (scores: ScoresDraft): TargetsScores => ({
  formats: scores.formats.flatMap(({ id, score }) =>
    Number(score) === 0 ? [] : [{ formatId: id, score: Number(score) }],
  ),
  ...totalsOf(scores),
});

export const matching = (scores: ScoresDraft, filter: string): readonly FormatScore[] => {
  const needle = filter.trim().toLocaleLowerCase();
  return scores.formats.filter(({ name }) => name.toLocaleLowerCase().includes(needle));
};

const formatsWord = (count: number) => (count === 1 ? "Custom Format" : "Custom Formats");

/**
 * What a Target's scores do, in words: a lead such as "Scores 2 of 5
 * Custom Formats. A Release needs 10 or more to be grabbed. ..." and a
 * warning for each total no Release can reach.
 */
export function scoresInWords(scores: ScoresDraft): {
  lead: string;
  warnings: readonly string[];
} {
  const values = scores.formats.map(({ score }) => read(score, scoreProblem, 0));
  const scored = values.filter((value) => value !== 0).length;
  const most = values.reduce((sum, value) => sum + Math.max(value, 0), 0);
  const { minimum, upgradeUntil, increment } = totalsOf(scores);
  const total = values.length;
  const lead =
    total === 0
      ? "No Custom Formats exist yet, so this Target ranks Releases by quality alone."
      : scored === 0
        ? "No Custom Format has a score here, so this Target ranks Releases by quality alone. Score a Custom Format to prefer the Releases that match it."
        : `Scores ${String(scored)} of ${String(total)} ${formatsWord(total)}. A Release needs a score of ${String(minimum)} or more to be grabbed. In the same tier, a Release replaces a file when it scores ${String(increment)} more, until the file scores ${String(upgradeUntil)}.`;
  const warnings = [
    ...(minimum > most
      ? [
          `No Release can reach the minimum of ${String(minimum)}, because the most a Release can score here is ${String(most)}. Lower the minimum or raise a score.`,
        ]
      : []),
    ...(upgradeUntil > most
      ? [
          `No file can reach the upgrade-until score of ${String(upgradeUntil)}, because the most a Release can score here is ${String(most)}. Files stay wanted past the cutoff. Lower upgrade-until or raise a score.`,
        ]
      : []),
  ];
  return { lead, warnings };
}
