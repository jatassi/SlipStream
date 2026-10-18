import { Link, Navigate, createFileRoute } from "@tanstack/react-router";

import { settingsSections } from "../../../settings/sections";
import { Page } from "../../../shell/page";
import { Chevron } from "../../../ui/chevron";
import { List, ListRow } from "../../../ui/list";
import { useShell } from "../../../ui/shell-context";

export const Route = createFileRoute("/_app/settings/")({
  staticData: { title: "Settings" },
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
            trailing={<Chevron />}
            render={<Link to="/settings/$section" params={{ section: section.id }} />}
          />
        ))}
      </List>
    </Page>
  );
}
