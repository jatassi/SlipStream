import type { ReactNode } from "react";

import type { HealthCategory } from "../api/gen/model";
import { Section } from "../ui/section";
import { SignalChip } from "../ui/signal-chip";
import { useCategoryCheck } from "./category-check";
import { type ItemProps, ItemRow } from "./item-row";
import { levels } from "./levels";
import { QuietLine } from "./quiet-line";
import { Unread } from "./unread";
import { useStableOrder } from "./use-stable-order";

export type CategoryProps = {
  category: HealthCategory;
  reading: ItemProps["reading"];
  /** What closes the category's section, such as Storage's settings. */
  end: ReactNode;
};

export function CategorySection({ category, reading, end }: CategoryProps) {
  if (category.level === null) {
    return <QuietLine category={category} end={end} />;
  }
  return <Problems category={category} level={category.level} reading={reading} end={end} />;
}

function Problems({
  category,
  level,
  reading,
  end,
}: CategoryProps & { level: NonNullable<HealthCategory["level"]> }) {
  const items = useStableOrder(category.items, (item) => item.key);
  const check = useCategoryCheck(category);
  return (
    <Section
      id={category.id}
      title={category.label}
      chip={<SignalChip kind={level}>{levels[level].chip}</SignalChip>}
      count={items.length}
      action={check.button}
    >
      {check.answer}
      <Unread category={category} />
      <div className="flex flex-col gap-2">
        {items.map((item) => (
          <ItemRow key={item.key} category={category.id} item={item} reading={reading} />
        ))}
      </div>
      {end}
    </Section>
  );
}
