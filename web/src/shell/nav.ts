import { Activity01Icon, LibraryIcon, ServerStack01Icon } from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";
import { useQuery } from "@tanstack/react-query";
import type { RegisteredRouter } from "@tanstack/react-router";

import type { WorksContentType } from "../api/gen/model";
import { getListContentTypesQueryKey, listContentTypes } from "../api/gen/query/works/works";
import type { CountKey } from "./counts";
import { contentTypeIcons } from "./icons";

type RoutePath = RegisteredRouter["routeTree"]["types"]["fileRouteTypes"]["to"];

export type AppPath = Exclude<RoutePath, `${string}$${string}`>;

export type NavId = "library" | "activity" | "this-node";

export type NavRow = {
  id: NavId;
  label: string;
  icon: IconSvgElement;
  to: AppPath;
  audience: "admin" | "member";
  place: "main" | "system";
  tab?: { order: number };
  count?: CountKey;
  children?: "content-types";
};

const contentTypeLibrary = "/library/$contentType";

export type NavChild = {
  id: string;
  label: string;
  icon: IconSvgElement;
  to: typeof contentTypeLibrary;
  params: { contentType: string };
};

export type ShownRow = Omit<NavRow, "children"> & { children: readonly NavChild[] };

export const adminNav: readonly NavRow[] = [
  {
    id: "library",
    label: "Library",
    icon: LibraryIcon,
    to: "/library",
    audience: "admin",
    place: "main",
    tab: { order: 1 },
    children: "content-types",
  },
  {
    id: "activity",
    label: "Activity",
    icon: Activity01Icon,
    to: "/activity",
    audience: "admin",
    place: "main",
    tab: { order: 2 },
    count: "activity",
  },
  {
    id: "this-node",
    label: "This Node",
    icon: ServerStack01Icon,
    to: "/system",
    audience: "admin",
    place: "system",
  },
];

export function useContentTypes(): readonly WorksContentType[] {
  return (
    useQuery({
      queryKey: getListContentTypesQueryKey(),
      queryFn: ({ signal }) => listContentTypes({ signal }),
    }).data?.contentTypes ?? []
  );
}

function useContentTypeChildren(): NavChild[] {
  return useContentTypes()
    .filter((type) => type.enabled)
    .map((type) => ({
      id: type.id,
      label: type.work.many,
      icon: contentTypeIcons[type.icon],
      to: contentTypeLibrary,
      params: { contentType: type.id },
    }));
}

export function useAdminNav(): ShownRow[] {
  const contentTypes = useContentTypeChildren();
  return adminNav.map((row) => ({
    ...row,
    children: row.children === "content-types" ? contentTypes : [],
  }));
}

export const tabRows = (rows: readonly ShownRow[]): ShownRow[] =>
  rows
    .filter((row) => row.tab !== undefined)
    .toSorted((a, b) => (a.tab?.order ?? 0) - (b.tab?.order ?? 0));

/** The paths a nav row or child leads to: the phone's top-level pages. */
export const navPaths: ReadonlySet<string> = new Set([
  ...adminNav.map((row) => row.to),
  contentTypeLibrary,
]);
