import { Link, Navigate, createFileRoute } from "@tanstack/react-router";

import { getGetSystemHealthSuspenseQueryOptions } from "../../../api/gen/query/health/health";
import { ensureQuery } from "../../../api/query-client";
import { SectionDot } from "../../../settings/section-alert";
import { settingsSections } from "../../../settings/sections";
import { Page } from "../../../shell/page";
import { Chevron } from "../../../ui/chevron";
import { List, ListRow } from "../../../ui/list";
import { useShell } from "../../../ui/shell-context";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/settings/")({
  staticData: { title: "Settings" },
  loader: ({ context }) =>
    ensureQuery(context.queryClient, getGetSystemHealthSuspenseQueryOptions()),
  pendingComponent: () => <Skeleton label="Loading Settings" />,
  component: SettingsTop,
});

// The desktop shell opens Settings on its first tab; on phone Settings is the
// list of sections, each a pushed page.
function SettingsTop() {
  if (useShell() === "desktop") {
    return (
      <Navigate to="/settings/$section" params={{ section: settingsSections[0].id }} replace />
    );
  }
  return (
    <Page>
      <List label="Settings">
        {settingsSections.map((section) => (
          <ListRow
            key={section.id}
            title={section.label}
            trailing={
              <span className="flex items-center gap-2">
                <SectionDot section={section} />
                <Chevron />
              </span>
            }
            render={<Link to="/settings/$section" params={{ section: section.id }} />}
          />
        ))}
      </List>
    </Page>
  );
}
