import { z } from "zod";

import { isApiError, isProblem, sentenceOf } from "../api/errors";
import type { TrackersGoal, TrackersRule, TrackersRuleBody } from "../api/gen/model";
import { type SubmitErrors, submitErrors } from "../ui/form-context";

export type Unit = "days" | "hours";

const unitSeconds: { readonly [U in Unit]: number } = { days: 86_400, hours: 3600 };

/** A ratio and a seeding time as the form edits them; "" is left out. */
export type Parts = { ratio: string; amount: string; unit: Unit };

export type Clears = TrackersRule["clears"];
export type Combine = TrackersGoal["combine"];
export type Action = NonNullable<TrackersGoal["action"]>;

export type RuleDraft = { clears: Clears; parts: Parts };
export type GoalDraft = { combine: Combine; parts: Parts; action: Action };

function partsOf(ratio: number | null, seedSeconds: number | null): Parts {
  const unit: Unit =
    seedSeconds === null || seedSeconds % unitSeconds.days === 0 ? "days" : "hours";
  return {
    ratio: ratio === null ? "" : String(ratio),
    amount: seedSeconds === null ? "" : String(seedSeconds / unitSeconds[unit]),
    unit,
  };
}

const decimal = (example: string) =>
  z
    .string()
    .trim()
    .regex(/^(?:\d+(?:\.\d+)?)?$/u, `Use a number, such as ${example}.`);

export const partsSchema = z.object({
  ratio: decimal("1.5"),
  amount: decimal("10"),
  unit: z.enum(["days", "hours"]),
});

const ratioOf = ({ ratio }: Parts): number | null => (ratio.trim() === "" ? null : Number(ratio));

const secondsOf = ({ amount, unit }: Parts): number | null =>
  amount.trim() === "" ? null : Math.round(Number(amount) * unitSeconds[unit]);

/** Which of the ratio and the seeding time a rule that clears each way counts. */
export const ruleCounts: {
  readonly [C in Clears]: { readonly ratio: boolean; readonly time: boolean };
} = {
  time: { ratio: false, time: true },
  ratio: { ratio: true, time: false },
  either: { ratio: true, time: true },
  both: { ratio: true, time: true },
};

export const ruleDraftOf = (rule: TrackersRule | null): RuleDraft =>
  rule === null
    ? { clears: "time", parts: partsOf(null, null) }
    : { clears: rule.clears, parts: partsOf(rule.ratio, rule.seedSeconds) };

export function ruleBodyOf({ clears, parts }: RuleDraft): TrackersRuleBody {
  const counts = ruleCounts[clears];
  return {
    clears,
    ratio: counts.ratio ? ratioOf(parts) : null,
    seedSeconds: counts.time ? secondsOf(parts) : null,
  };
}

export const goalDraftOf = (goal: TrackersGoal): GoalDraft => ({
  combine: goal.combine,
  parts: partsOf(goal.ratio, goal.seedSeconds),
  action: goal.action ?? "remove-delete",
});

export const goalBodyOf = ({ combine, parts, action }: GoalDraft): TrackersGoal =>
  combine === "forever"
    ? { combine, ratio: null, seedSeconds: null, action: null }
    : { combine, ratio: ratioOf(parts), seedSeconds: secondsOf(parts), action };

const formFields: Readonly<Record<string, string>> = {
  "body.ratio": "parts.ratio",
  "body.seedSeconds": "parts.amount",
};

/** A refused save: seeding_value_refused on the field it names, anything else for the whole form. */
export function seedingRefusal(error: unknown): SubmitErrors {
  if (isProblem(error, "seeding_value_refused")) {
    const fields = Object.fromEntries(
      error.errors.flatMap(({ location, message }) => {
        const field = formFields[location];
        return field === undefined ? [] : [[field, message]];
      }),
    );
    return Object.keys(fields).length === 0 ? { form: sentenceOf(error), fields } : { fields };
  }
  return isApiError(error) ? { form: sentenceOf(error), fields: {} } : submitErrors(error);
}
