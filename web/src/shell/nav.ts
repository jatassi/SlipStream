import {
  Activity01Icon,
  Home01Icon,
  LibraryIcon,
  Search01Icon,
  ServerStack01Icon,
  Settings01Icon,
  Task01Icon,
  TimeScheduleIcon,
  UserCircleIcon,
  UserGroupIcon,
} from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";
import { useQuery } from "@tanstack/react-query";
import type { RegisteredRouter } from "@tanstack/react-router";

import type { WorksContentType } from "../api/gen/model";
import { getListContentTypesQueryKey, listContentTypes } from "../api/gen/query/works/works";
import type { CountKey } from "./counts";
import { contentTypeIcons } from "./icons";

type RoutePath = RegisteredRouter["routeTree"]["types"]["fileRouteTypes"]["to"];

export type AppPath = Exclude<RoutePath, `${string}$${string}`>;

export type NavId =
  | "library"
  | "activity"
  | "this-node"
  | "accounts"
  | "settings"
  | "tasks"
  | "home"
  | "search"
  | "requests"
  | "account";

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
const settingsSection = "/settings/$section";

export type NavChild = {
  id: string;
  label: string;
  icon: IconSvgElement;
  to: typeof contentTypeLibrary;
  params: { contentType: string };
};

export type ShownRow = Omit<NavRow, "children"> & { children: readonly NavChild[] };

export type NavTable = readonly [NavRow, ...NavRow[]];

export const adminNav: NavTable = [
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
  {
    id: "accounts",
    label: "Accounts",
    icon: UserGroupIcon,
    to: "/accounts",
    audience: "admin",
    place: "system",
  },
  {
    id: "settings",
    label: "Settings",
    icon: Settings01Icon,
    to: "/settings",
    audience: "admin",
    place: "system",
  },
  {
    id: "tasks",
    label: "Tasks",
    icon: TimeScheduleIcon,
    to: "/tasks",
    audience: "admin",
    place: "system",
  },
];

export const portalNav: NavTable = [
  {
    id: "home",
    label: "Home",
    icon: Home01Icon,
    to: "/portal",
    audience: "member",
    place: "main",
    tab: { order: 1 },
  },
  {
    id: "search",
    label: "Search",
    icon: Search01Icon,
    to: "/portal/search",
    audience: "member",
    place: "main",
    tab: { order: 2 },
  },
  {
    id: "requests",
    label: "Requests",
    icon: Task01Icon,
    to: "/portal/requests",
    audience: "member",
    place: "main",
    tab: { order: 3 },
  },
  {
    id: "account",
    label: "Account",
    icon: UserCircleIcon,
    to: "/portal/account",
    audience: "member",
    place: "main",
    tab: { order: 4 },
  },
];

export const portalRows: readonly ShownRow[] = portalNav.map((row) => ({ ...row, children: [] }));

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
  ...portalNav.map((row) => row.to),
  contentTypeLibrary,
  settingsSection,
]);
