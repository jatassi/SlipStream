import type { HealthCategory, HealthItemLevel } from "../api/gen/model";

export const levels: {
  readonly [L in HealthItemLevel]: {
    readonly chip: string;
    readonly part: "failure" | "warning";
    readonly rank: number;
  };
} = {
  fail: { chip: "Error", part: "failure", rank: 0 },
  warn: { chip: "Warning", part: "warning", rank: 1 },
};

const quietRank = 2;

const rankOf = ({ level }: HealthCategory) => (level === null ? quietRank : levels[level].rank);

/** Categories with errors first, then warnings, then the rest, each kept in the order given. */
export const worstFirst = (categories: readonly HealthCategory[]): HealthCategory[] =>
  categories.toSorted((a, b) => rankOf(a) - rankOf(b));

export const itemAnchor = (category: string, key: string): string => `health-${category}-${key}`;
