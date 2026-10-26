import { getGetContentTypeRefreshSuspenseQueryOptions } from "../api/gen/query/metadata/metadata";
import { getListRootsSuspenseQueryOptions } from "../api/gen/query/roots/roots";
import { getListProfilesSuspenseQueryOptions } from "../api/gen/query/targets/targets";
import {
  getGetContentTypeSettingsSuspenseQueryOptions,
  getListContentTypesSuspenseQueryOptions,
  useListContentTypesSuspense,
} from "../api/gen/query/works/works";
import { ensureQuery, freshQuery } from "../api/query-client";
import { ContentTypeRow } from "./content-type-row";
import type { SettingsSection } from "./section";

function ContentTypes() {
  const { contentTypes } = useListContentTypesSuspense().data;
  return (
    <div className="flex flex-col gap-3">
      {contentTypes.map((type) => (
        <ContentTypeRow key={type.id} type={type} />
      ))}
    </div>
  );
}

export const contentTypesSection: SettingsSection = {
  id: "content-types",
  label: "Content Types",
  load: async (queryClient) => {
    const { contentTypes } = await ensureQuery(
      queryClient,
      getListContentTypesSuspenseQueryOptions(),
    );
    await Promise.all([
      ensureQuery(queryClient, getListProfilesSuspenseQueryOptions()),
      ensureQuery(queryClient, getListRootsSuspenseQueryOptions()),
      ...contentTypes.flatMap((type) =>
        type.enabled
          ? [
              freshQuery(queryClient, getGetContentTypeSettingsSuspenseQueryOptions(type.id)),
              freshQuery(queryClient, getGetContentTypeRefreshSuspenseQueryOptions(type.id)),
            ]
          : [],
      ),
    ]);
  },
  component: ContentTypes,
};
