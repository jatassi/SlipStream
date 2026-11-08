import { Link, Outlet, useMatch } from "@tanstack/react-router";

import { useGetTrackerSuspense } from "../api/gen/query/trackers/trackers";
import { Page } from "../shell/page";
import { PageTabs, pageTab } from "../ui/page-tabs";
import { useShell } from "../ui/shell-context";
import { PrivacyChip, TrackerHeader } from "./header";
import { SectionCount, SectionRows } from "./section-rows";
import { trackerSections } from "./sections";
import { TrackerActions } from "./tracker-actions";
import { useTracker } from "./tracker";

// On phone each section is its own page: the Tracker's top page carries its
// header, lists the other sections and then its torrents, as a Node's does.
export function TrackerLayout({ trackerId }: { trackerId: string }) {
  const tracker = useTracker(trackerId);
  const page = useGetTrackerSuspense(trackerId).data;
  const menu = <TrackerActions tracker={tracker} />;
  const beside = <PrivacyChip tracker={tracker} />;
  const top =
    useMatch({ from: "/_app/settings/trackers/$trackerId/_torrents/", shouldThrow: false }) !==
    undefined;
  const inTorrents =
    useMatch({ from: "/_app/settings/trackers/$trackerId/_torrents", shouldThrow: false }) !==
    undefined;
  if (useShell() === "phone") {
    return (
      <Page {...(top && { action: menu, beside })}>
        {top && <TrackerHeader tracker={tracker} page={page} />}
        {top && <SectionRows tracker={tracker} />}
        <Outlet />
      </Page>
    );
  }
  return (
    <Page title={tracker.name} beside={beside} action={menu}>
      <TrackerHeader tracker={tracker} page={page} />
      <PageTabs label={tracker.name}>
        {trackerSections.map((section) => (
          <Link
            key={section.id}
            to={section.to}
            params={{ trackerId }}
            activeOptions={{ exact: true }}
            aria-current={section.id === "torrents" && inTorrents ? "page" : undefined}
            className={`${pageTab} gap-2`}
          >
            {section.label}
            <SectionCount section={section} tracker={tracker} />
          </Link>
        ))}
      </PageTabs>
      <Outlet />
    </Page>
  );
}
