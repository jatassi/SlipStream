import type { HealthCategoryID, HealthItem, HealthReport } from "../api/gen/model";
import type { AlertTone } from "../ui/count";

export type SectionHealth = {
  level: AlertTone;
  /** The counts a screen reader hears, such as "1 error, 2 warnings". */
  words: string;
  /** The worst item, the one the section opens with. */
  first: { category: HealthCategoryID; item: HealthItem };
};

const nouns = { fail: ["error", "errors"], warn: ["warning", "warnings"] } as const;

/** The worst level, the counts and the worst item among a settings section's categories; undefined while they have none. */
export function sectionHealth(
  report: HealthReport,
  ids: readonly HealthCategoryID[],
): SectionHealth | undefined {
  const items = report.categories
    .filter((category) => ids.includes(category.id))
    .flatMap((category) => category.items.map((item) => ({ category: category.id, item })));
  const first = items.find(({ item }) => item.level === "fail") ?? items[0];
  if (first === undefined) {
    return undefined;
  }
  const words = (["fail", "warn"] as const)
    .map((level) => {
      const count = items.filter(({ item }) => item.level === level).length;
      const [one, many] = nouns[level];
      return count === 0 ? undefined : `${count} ${count === 1 ? one : many}`;
    })
    .filter((part) => part !== undefined)
    .join(", ");
  return { level: first.item.level, words, first };
}
