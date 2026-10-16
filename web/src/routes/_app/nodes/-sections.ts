import {
  Globe02Icon,
  InformationCircleIcon,
  Settings02Icon,
  TextAlignLeftIcon,
} from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";

import type { RoutePath } from "../../../shell/nav";

export type NodeSection = {
  readonly id: string;
  readonly label: string;
  readonly icon: IconSvgElement;
  readonly to: Extract<RoutePath, `/nodes/$nodeId${string}`>;
};

/**
 * A Node's sections, in order. They draw the desktop page tabs, the phone
 * Node page's rows and the palette's Go to commands. A later per-Node screen
 * joins as a row here.
 */
export const nodeSections = [
  { id: "overview", label: "Overview", icon: InformationCircleIcon, to: "/nodes/$nodeId" },
  { id: "web", label: "Web and API", icon: Globe02Icon, to: "/nodes/$nodeId/web" },
  { id: "logs", label: "Logs", icon: TextAlignLeftIcon, to: "/nodes/$nodeId/logs" },
  {
    id: "log-settings",
    label: "Log settings",
    icon: Settings02Icon,
    to: "/nodes/$nodeId/log-settings",
  },
] as const satisfies readonly NodeSection[];

export type NodeSectionId = (typeof nodeSections)[number]["id"];

export function sectionOf(id: NodeSectionId): NodeSection {
  const section = nodeSections.find((candidate) => candidate.id === id);
  if (section === undefined) {
    throw new Error(`no Node section ${id}`);
  }
  return section;
}
