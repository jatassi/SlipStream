import type { ReactNode } from "react";

import type { HealthCategoryID, HealthReport } from "../api/gen/model";
import { Page } from "../shell/page";
import { useNow } from "../tasks/use-now";
import { CategorySection } from "./category-section";
import { HealthHead } from "./health-head";
import { worstFirst } from "./levels";
import { useStableOrder } from "./use-stable-order";

const sinceTick = 60_000;

const ends: { readonly [C in HealthCategoryID]?: ReactNode } = {};

export function HealthPage({ report }: { report: HealthReport }) {
  const categories = useStableOrder(worstFirst(report.categories), (category) => category.id);
  const reading = { now: useNow(sinceTick), zone: undefined };
  return (
    <Page>
      <HealthHead report={report} />
      {categories.map((category) => (
        <CategorySection
          key={category.id}
          category={category}
          reading={reading}
          end={ends[category.id]}
        />
      ))}
    </Page>
  );
}
