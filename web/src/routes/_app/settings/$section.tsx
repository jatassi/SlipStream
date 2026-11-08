import { createFileRoute } from "@tanstack/react-router";

import { getGetSystemHealthSuspenseQueryOptions } from "../../../api/gen/query/health/health";
import { ensureQuery } from "../../../api/query-client";
import { SectionWarning } from "../../../settings/section-alert";
import { sectionOf } from "../../../settings/sections";
import { SettingsNav } from "../../../settings/settings-nav";
import { Page } from "../../../shell/page";
import { loaderTitle } from "../../../shell/titles";
import { useShell } from "../../../ui/shell-context";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/settings/$section")({
  staticData: { title: loaderTitle },
  // A Release's "Open in tester" names it, for the Custom Formats tester to read.
  validateSearch: (search: Record<string, unknown>): { release?: string } =>
    typeof search["release"] === "string" ? { release: search["release"] } : {},
  loader: async ({ context, params }) => {
    const section = sectionOf(params.section);
    await Promise.all([
      section.load(context.queryClient),
      ensureQuery(context.queryClient, getGetSystemHealthSuspenseQueryOptions()),
    ]);
    return { title: section.label, trail: [{ title: "Settings", pathname: "/settings" }] };
  },
  pendingComponent: () => <Skeleton label="Loading Settings" />,
  component: SettingsPage,
});

// On phone each section is a pushed page under the Settings list, so only
// the desktop shell draws the nav.
function SettingsPage() {
  const { section: id } = Route.useParams();
  const section = sectionOf(id);
  const body = (
    <div
      className={`flex min-w-0 flex-1 flex-col gap-6 ${section.wide === true ? "" : "max-w-[560px]"}`}
    >
      <SectionWarning section={section} />
      <section.component key={id} />
    </div>
  );
  if (useShell() === "phone") {
    return <Page>{body}</Page>;
  }
  return (
    <Page title="Settings">
      <div className="flex items-start gap-8">
        <SettingsNav />
        {body}
      </div>
    </Page>
  );
}
