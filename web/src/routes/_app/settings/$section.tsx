import { Link, createFileRoute } from "@tanstack/react-router";

import { settingsSections, sectionOf } from "../../../settings/sections";
import { Page } from "../../../shell/page";
import { loaderTitle } from "../../../shell/titles";
import { PageTabs, pageTab } from "../../../ui/page-tabs";
import { useShell } from "../../../ui/shell-context";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/settings/$section")({
  staticData: { title: loaderTitle },
  loader: async ({ context, params }) => {
    const section = sectionOf(params.section);
    await section.load(context.queryClient);
    return { title: section.label, trail: [{ title: "Settings", pathname: "/settings" }] };
  },
  pendingComponent: () => <Skeleton label="Loading Settings" />,
  component: SettingsPage,
});

// On phone each section is a pushed page under the Settings list, so only
// the desktop shell draws the tabs.
function SettingsPage() {
  const { section: id } = Route.useParams();
  const section = sectionOf(id);
  const body = (
    <div className={`flex flex-col gap-6 ${section.wide === true ? "" : "max-w-[560px]"}`}>
      <section.component key={id} />
    </div>
  );
  if (useShell() === "phone") {
    return <Page>{body}</Page>;
  }
  return (
    <Page title="Settings">
      <PageTabs label="Settings">
        {settingsSections.map((row) => (
          <Link
            key={row.id}
            to="/settings/$section"
            params={{ section: row.id }}
            className={pageTab}
          >
            {row.label}
          </Link>
        ))}
      </PageTabs>
      {body}
    </Page>
  );
}
