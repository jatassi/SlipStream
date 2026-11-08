import { createFileRoute } from "@tanstack/react-router";

import {
  getCustomFormatSchemaSuspenseQueryOptions,
  useCustomFormatSchemaSuspense,
} from "../../../api/gen/query/formats/formats";
import { ensureQuery } from "../../../api/query-client";
import { NewFormatEditor } from "../../../formats/format-pages";
import { Page } from "../../../shell/page";
import { loaderTitle } from "../../../shell/titles";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/settings/custom-formats/new")({
  staticData: { title: loaderTitle },
  loader: async ({ context }) => {
    await ensureQuery(context.queryClient, getCustomFormatSchemaSuspenseQueryOptions());
    return {
      title: "New Custom Format",
      trail: [
        { title: "Settings", pathname: "/settings" },
        { title: "Custom Formats", pathname: "/settings/custom-formats" },
      ],
    };
  },
  pendingComponent: () => <Skeleton label="Loading the Custom Format editor" />,
  component: NewFormatPage,
});

function NewFormatPage() {
  const schema = useCustomFormatSchemaSuspense().data;
  return (
    <Page>
      <NewFormatEditor schema={schema} />
    </Page>
  );
}
