import { createFileRoute } from "@tanstack/react-router";

import {
  getCustomFormatSchemaSuspenseQueryOptions,
  getGetCustomFormatSuspenseQueryOptions,
  getListCustomFormatsSuspenseQueryOptions,
  useCustomFormatSchemaSuspense,
  useGetCustomFormatSuspense,
  useListCustomFormatsSuspense,
} from "../../../api/gen/query/formats/formats";
import { ensureQuery, freshQuery } from "../../../api/query-client";
import { SavedFormatEditor } from "../../../formats/format-pages";
import { Page } from "../../../shell/page";
import { loaderTitle } from "../../../shell/titles";
import { Skeleton } from "../../../ui/skeleton";

export const Route = createFileRoute("/_app/settings/custom-formats/$formatId")({
  staticData: { title: loaderTitle },
  loader: async ({ context, params: { formatId } }) => {
    const [format] = await Promise.all([
      freshQuery(context.queryClient, getGetCustomFormatSuspenseQueryOptions(formatId)),
      ensureQuery(context.queryClient, getCustomFormatSchemaSuspenseQueryOptions()),
      ensureQuery(context.queryClient, getListCustomFormatsSuspenseQueryOptions()),
    ]);
    return {
      title: format.name,
      trail: [
        { title: "Settings", pathname: "/settings" },
        { title: "Custom Formats", pathname: "/settings/custom-formats" },
      ],
    };
  },
  pendingComponent: () => <Skeleton label="Loading the Custom Format" />,
  component: FormatPage,
});

function FormatPage() {
  const { formatId } = Route.useParams();
  const format = useGetCustomFormatSuspense(formatId).data;
  const schema = useCustomFormatSchemaSuspense().data;
  const { matchLimitMilliseconds } = useListCustomFormatsSuspense().data;
  return (
    <Page>
      <SavedFormatEditor
        key={format.id}
        format={format}
        schema={schema}
        matchLimit={matchLimitMilliseconds}
      />
    </Page>
  );
}
