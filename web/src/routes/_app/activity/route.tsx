import {
  Link,
  Outlet,
  createFileRoute,
  useChildMatches,
  useNavigate,
} from "@tanstack/react-router";
import { z } from "zod";

import { Page, useShellSlots } from "../../../shell/page";
import { PageTabs, pageTab } from "../../../ui/page-tabs";
import { Segmented } from "../../../ui/segmented";
import { useShell } from "../../../ui/shell-context";

const activityViews = [
  { id: "jobs", label: "Jobs", to: "/activity", route: "/_app/activity/" },
  { id: "torrents", label: "Torrents", to: "/activity/torrents", route: "/_app/activity/torrents" },
] as const;

// Shared by both views, so a Node, Content Type or search survives a switch.
const activitySearch = z.object({
  node: z.uuid().optional().catch(undefined),
  type: z.string().optional().catch(undefined),
  q: z.string().optional().catch(undefined),
});

type ActivitySearch = z.infer<typeof activitySearch>;

const shared = ({ node, type, q }: ActivitySearch): ActivitySearch => ({ node, type, q });

export const Route = createFileRoute("/_app/activity")({
  staticData: { title: "Activity" },
  validateSearch: activitySearch,
  component: ActivityLayout,
});

function ActivityLayout() {
  const pushed = useShellSlots().heading.kind === "bar";
  if (useShell() === "phone") {
    return (
      <Page title="Activity" beside={pushed ? undefined : <PhoneViews />}>
        <Outlet />
      </Page>
    );
  }
  return (
    <Page title="Activity">
      <PageTabs label="Activity">
        {activityViews.map((view) => (
          <Link
            key={view.id}
            to={view.to}
            search={shared}
            activeOptions={{ exact: view.id === "jobs" }}
            className={pageTab}
          >
            {view.label}
          </Link>
        ))}
      </PageTabs>
      <Outlet />
    </Page>
  );
}

function PhoneViews() {
  const navigate = useNavigate();
  const shown = useChildMatches({ select: (matches) => matches[0]?.routeId });
  const current = activityViews.find((view) => view.route === shown);
  if (current === undefined) {
    throw new Error("Activity rendered outside its views");
  }
  return (
    <Segmented
      label="View"
      options={activityViews.map(({ to, label }) => ({ value: to, label }))}
      value={current.to}
      onValueChange={(to) => {
        void navigate({ to, search: shared });
      }}
    />
  );
}
