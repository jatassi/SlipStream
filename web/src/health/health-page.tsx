import type { ReactNode } from "react";

import type { HealthCategoryID, HealthReport } from "../api/gen/model";
import { Page } from "../shell/page";
import { CategorySection } from "./category-section";
import { HealthHead } from "./health-head";
import { worstFirst } from "./levels";
import { StorageSettings } from "./storage-settings";
import { useStableOrder } from "./use-stable-order";

const ends: { readonly [C in HealthCategoryID]?: ReactNode } = {
  storage: (
    <div className="mt-3">
      <StorageSettings toLibraryRoots />
    </div>
  ),
};

export function HealthPage({ report }: { report: HealthReport }) {
  const categories = useStableOrder(worstFirst(report.categories), (category) => category.id);
  return (
    <Page>
      <HealthHead report={report} />
      {categories.map((category) => (
        <CategorySection key={category.id} category={category} end={ends[category.id]} />
      ))}
    </Page>
  );
}
