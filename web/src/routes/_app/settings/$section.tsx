import { createFileRoute } from "@tanstack/react-router";

import { settingsSections, sectionOf } from "../../../settings/sections";
import { Page } from "../../../shell/page";
import { loaderTitle } from "../../../shell/titles";
import { Segmented } from "../../../ui/segmented";
import { Skeleton } from "../../../ui/skeleton";

const sectionOptions = settingsSections.map(({ id, label }) => ({ value: id, label }));

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

function SettingsPage() {
  const { section } = Route.useParams();
  const navigate = Route.useNavigate();
  const Section = sectionOf(section).component;
  return (
    <Page>
      <div className="flex max-w-[560px] flex-col gap-6">
        <Segmented
          label="Settings section"
          options={sectionOptions}
          value={section}
          onValueChange={(next) => {
            void navigate({ params: { section: next } });
          }}
        />
        <Section key={section} />
      </div>
    </Page>
  );
}
