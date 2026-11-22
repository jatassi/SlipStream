import { ArrowRight01Icon } from "@hugeicons/core-free-icons";
import { Link } from "@tanstack/react-router";

import type { HealthReport } from "../api/gen/model";
import { HealthVerdict } from "../health/health-head";
import { Icon } from "../ui/icon";
import { List, ListRow } from "../ui/list";
import { Section } from "../ui/section";
import { categoryWords, healthAnchor } from "./words";

/** Health's verdict, then each category with a problem or a check that couldn't run, linking to it on the Health page. */
export function Health({ report }: { report: HealthReport }) {
  const troubled = report.categories.filter(
    ({ level, unread }) => level !== null || unread !== undefined,
  );
  return (
    <Section id={healthAnchor} title="Health">
      <div className="flex flex-col gap-3">
        <HealthVerdict report={report} at="/health" scale="section" />
        {troubled.length > 0 && (
          <List label="Health categories with a problem">
            {troubled.map((category) => (
              <ListRow
                key={category.id}
                title={
                  category.level === null
                    ? category.label
                    : `${category.label} · ${categoryWords(category)}`
                }
                {...(category.unread !== undefined && {
                  meta: `SlipStream couldn't check ${category.phrase}. It shows what the last successful check found.`,
                })}
                trailing={<Icon icon={ArrowRight01Icon} className="size-4" />}
                render={<Link to="/health" hash={category.id} preload={false} />}
              />
            ))}
          </List>
        )}
      </div>
    </Section>
  );
}
