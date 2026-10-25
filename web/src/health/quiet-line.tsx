import type { ReactNode } from "react";

import type { HealthCategory } from "../api/gen/model";
import { Icon } from "../ui/icon";
import { useShell } from "../ui/shell-context";
import { signals } from "../ui/signal-chip";
import { useCategoryCheck } from "./category-check";
import { Unread } from "./unread";

const gaps = {
  desktop: "[section+&]:mt-6 [[data-quiet]+&]:mt-1",
  phone: "[section+&]:mt-7 [[data-quiet]+&]:mt-1",
} as const;

/** A category with nothing wrong: one line, with its Check now. */
export function QuietLine({ category, end }: { category: HealthCategory; end: ReactNode }) {
  const check = useCategoryCheck(category);
  return (
    <section
      id={category.id}
      aria-label={category.label}
      data-quiet
      className={`scroll-mt-20 ${gaps[useShell()]}`}
    >
      <div className="flex min-h-(--control-height) flex-wrap items-center gap-x-3 gap-y-1">
        <Icon icon={signals.ok.icon} className="size-4 shrink-0 text-ok" />
        <p className="text-ink-2">Nothing is wrong with {category.phrase}.</p>
        {check.button}
      </div>
      {check.answer}
      <Unread category={category} />
      {end}
    </section>
  );
}
