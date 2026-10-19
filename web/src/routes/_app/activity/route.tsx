import {
  Link,
  Outlet,
  createFileRoute,
  useChildMatches,
  useNavigate,
} from "@tanstack/react-router";

import { Page } from "../../../shell/page";
import { PageTabs, pageTab } from "../../../ui/page-tabs";
import { Segmented } from "../../../ui/segmented";
import { useShell } from "../../../ui/shell-context";

const activityViews = [
  { id: "jobs", label: "Jobs", to: "/activity", route: "/_app/activity/" },
  { id: "torrents", label: "Torrents", to: "/activity/torrents", route: "/_app/activity/torrents" },
] as const;

export const Route = createFileRoute("/_app/activity")({
  staticData: { title: "Activity" },
  component: ActivityLayout,
});

function ActivityLayout() {
  if (useShell() === "phone") {
    return (
      <Page title="Activity" beside={<PhoneViews />}>
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
        void navigate({ to });
      }}
    />
  );
}
