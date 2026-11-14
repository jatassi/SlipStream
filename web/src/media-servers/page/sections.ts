import {
  ArrowDataTransferVerticalIcon,
  FolderLinksIcon,
  InformationCircleIcon,
  PlayCircle02Icon,
  Settings02Icon,
} from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";

import type { MediaserversServerView as Server } from "../../api/gen/model";
import type { AlertTone } from "../../ui/count";

export type ServerSection = {
  readonly id: string;
  readonly label: string;
  readonly icon: IconSvgElement;
  readonly to: `/settings/media-servers/$serverId${string}`;
  /** The number beside the section's tab or row, shown above zero. */
  readonly count?: (server: Server) => number;
  readonly dot?: (server: Server) => AlertTone | undefined;
};

/**
 * A server page's sections, in order. They draw the desktop page tabs and
 * the phone page's rows. The first is the page's index.
 */
export const serverSections = [
  {
    id: "playing",
    label: "Now playing",
    icon: PlayCircle02Icon,
    to: "/settings/media-servers/$serverId",
  },
  {
    id: "changes",
    label: "Changes",
    icon: ArrowDataTransferVerticalIcon,
    to: "/settings/media-servers/$serverId/changes",
    dot: (server) => (server.sends === "failing" ? "fail" : undefined),
  },
  {
    id: "mappings",
    label: "Path mappings",
    icon: FolderLinksIcon,
    to: "/settings/media-servers/$serverId/mappings",
    count: (server) => server.mappingsNeeded,
    dot: (server) => (server.mappingsNeeded > 0 ? "warn" : undefined),
  },
  {
    id: "settings",
    label: "Settings",
    icon: Settings02Icon,
    to: "/settings/media-servers/$serverId/settings",
  },
  {
    id: "details",
    label: "Details",
    icon: InformationCircleIcon,
    to: "/settings/media-servers/$serverId/details",
  },
] as const satisfies readonly ServerSection[];

export type ServerSectionId = (typeof serverSections)[number]["id"];

export function serverSectionOf(id: ServerSectionId): ServerSection {
  const section = serverSections.find((candidate) => candidate.id === id);
  if (section === undefined) {
    throw new Error(`no Media Server section ${id}`);
  }
  return section;
}
