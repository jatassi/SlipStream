import { Link, Outlet, useMatch } from "@tanstack/react-router";

import { useGetMediaServerSuspense } from "../../api/gen/query/mediaservers/mediaservers";
import { Page } from "../../shell/page";
import { PageTabs, pageTab } from "../../ui/page-tabs";
import { useShell } from "../../ui/shell-context";
import { ServerHeader } from "./header";
import { RemoveServer } from "./remove-server";
import { SectionCount, SectionRows } from "./section-rows";
import { serverSections } from "./sections";

/** A Media Server's page: what it needs and Test, then its sections as page tabs, or rows on phone. */
export function ServerLayout({ serverId }: { serverId: string }) {
  const server = useGetMediaServerSuspense(serverId).data;
  const action = <RemoveServer server={server} />;
  const top =
    useMatch({ from: "/_app/settings/media-servers/$serverId/", shouldThrow: false }) !== undefined;
  if (useShell() === "phone") {
    return (
      <Page {...(top && { action })}>
        {top && <ServerHeader server={server} />}
        {top && <SectionRows server={server} />}
        <Outlet />
      </Page>
    );
  }
  return (
    <Page title={server.name} action={action}>
      <ServerHeader server={server} />
      <PageTabs label={server.name}>
        {serverSections.map((section) => (
          <Link
            key={section.id}
            to={section.to}
            params={{ serverId }}
            activeOptions={{ exact: true }}
            className={`${pageTab} gap-2`}
          >
            {section.label}
            <SectionCount section={section} server={server} />
          </Link>
        ))}
      </PageTabs>
      <Outlet />
    </Page>
  );
}
