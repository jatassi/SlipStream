import { Navigate, createFileRoute } from "@tanstack/react-router";

import { getGetSystemHealthSuspenseQueryOptions } from "../../../api/gen/query/health/health";
import { ensureQuery } from "../../../api/query-client";
import { settingsGroups } from "../../../settings/sections";
import { SettingsList } from "../../../settings/settings-list";
import { Page } from "../../../shell/page";
import { useShell } from "../../../ui/shell-context";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/settings/")({
  staticData: { title: "Settings" },
  loader: ({ context }) =>
    ensureQuery(context.queryClient, getGetSystemHealthSuspenseQueryOptions()),
  pendingComponent: () => <Skeleton label="Loading Settings" />,
  component: SettingsTop,
});

// The desktop shell opens Settings on its first section; on phone Settings
// lists its sections by group, each a pushed page.
function SettingsTop() {
  if (useShell() === "desktop") {
    return (
      <Navigate
        to="/settings/$section"
        params={{ section: settingsGroups[0].sections[0].id }}
        replace
      />
    );
  }
  return (
    <Page>
      <SettingsList />
    </Page>
  );
}
