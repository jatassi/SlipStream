import { type ConditionDraft, excepts, pickedWords } from "./draft";

const subject = (condition: ConditionDraft): string => condition.kind.words.toLowerCase();

const is = (negate: boolean) => (negate ? "isn't" : "is");

/** One condition as a clause, such as `its source is WEB-DL`; undefined while its value is blank. */
export function clauseOf(condition: ConditionDraft): string | undefined {
  const { value, negate } = condition;
  switch (value.shape) {
    case "pattern": {
      const text = value.text.trim();
      return text === ""
        ? undefined
        : `its ${subject(condition)} ${negate ? "doesn't match" : "matches"} "${text}"`;
    }
    case "pick": {
      const words = pickedWords(condition.kind, value.picked);
      if (excepts(condition.kind) && value.except) {
        return negate ? `its only language is ${words}` : `it has a language other than ${words}`;
      }
      return `its ${subject(condition)} ${is(negate)} ${words}`;
    }
    case "range": {
      const [min, max] = [value.min.trim(), value.max.trim()];
      if (min === "" || max === "") {
        return undefined;
      }
      return condition.kind.kind === "SizeSpecification"
        ? `its size ${is(negate)} over ${min} GB and at most ${max} GB`
        : `its ${subject(condition)} ${is(negate)} ${min} to ${max}`;
    }
    default:
      return value satisfies never;
  }
}

/** "a or b", "a, b, or c", and between groups "a, and b". */
function joined(items: readonly string[], word: "and" | "or", comma: boolean): string {
  const rest = items.slice(0, -1);
  const last = items.at(-1) ?? "";
  if (rest.length === 0) {
    return last;
  }
  return `${rest.join(", ")}${comma || rest.length > 1 ? "," : ""} ${word} ${last}`;
}

/**
 * A kind's conditions as Radarr and Sonarr combine them: every required
 * one must match, and with none required any one may. A condition that
 * isn't required says nothing beside a required one of its kind.
 */
function groupOf(conditions: readonly ConditionDraft[]): string | undefined {
  const required = conditions.filter((condition) => condition.required);
  const counted = required.length > 0 ? required : conditions;
  const clauses = counted.flatMap((condition) => clauseOf(condition) ?? []);
  if (clauses.length === 0) {
    return undefined;
  }
  return joined(clauses, required.length > 0 ? "and" : "or", false);
}

/**
 * Which Releases a format matches, in words, such as
 * `Matches a Release when its release title matches "DV", and its source is WEB-DL.`
 * Undefined while no condition has a value.
 */
export function matchSentence(conditions: readonly ConditionDraft[]): string | undefined {
  const kinds = [...new Set(conditions.map((condition) => condition.kind.kind))];
  const groups = kinds.flatMap(
    (kind) => groupOf(conditions.filter((condition) => condition.kind.kind === kind)) ?? [],
  );
  if (groups.length === 0) {
    return undefined;
  }
  return `Matches a Release when ${joined(groups, "and", true)}.`;
}

/** The slow-pattern warning: a match ran past the limit on `releases` Releases since the last save. */
export function slowSentence(releases: number, limitMilliseconds: number): string {
  const counted = releases === 1 ? "1 Release" : `${String(releases)} Releases`;
  const those = releases === 1 ? "That Release counts" : "Those Releases count";
  return `A pattern ran past its ${String(limitMilliseconds)} ms limit on ${counted} since this Custom Format was saved. ${those} as not matching it. Shorten or simplify the pattern.`;
}
