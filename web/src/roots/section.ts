import { getListNodesSuspenseQueryOptions } from "../api/gen/query/nodes/nodes";
import {
  getGetRootSettingsSuspenseQueryOptions,
  getListRootsSuspenseQueryOptions,
  getListVolumesSuspenseQueryOptions,
} from "../api/gen/query/roots/roots";
import { getListContentTypesSuspenseQueryOptions } from "../api/gen/query/works/works";
import { ensureQuery, freshQuery } from "../api/query-client";
import type { SettingsSection } from "../settings/section";
import { LibraryRootsSection } from "./library-roots-section";

export const libraryRootsSection: SettingsSection = {
  id: "library-roots",
  label: "Library roots",
  load: async (queryClient) => {
    await Promise.all([
      ensureQuery(queryClient, getListRootsSuspenseQueryOptions()),
      ensureQuery(queryClient, getListVolumesSuspenseQueryOptions()),
      ensureQuery(queryClient, getListContentTypesSuspenseQueryOptions()),
      ensureQuery(queryClient, getListNodesSuspenseQueryOptions()),
      freshQuery(queryClient, getGetRootSettingsSuspenseQueryOptions()),
    ]);
  },
  component: LibraryRootsSection,
  wide: true,
};
