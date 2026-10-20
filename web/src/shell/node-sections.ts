import {
  Download04Icon,
  Globe02Icon,
  InformationCircleIcon,
  Layers01Icon,
  Settings02Icon,
  TextAlignLeftIcon,
} from "@hugeicons/core-free-icons";
import type { IconSvgElement } from "@hugeicons/react";
import { useQuery } from "@tanstack/react-query";

import type { NodesNode, NodesRoleViewRole } from "../api/gen/model";
import { getListNodesQueryKey, listNodes } from "../api/gen/query/nodes/nodes";
import type { Command } from "./commands";
import type { RoutePath } from "./nav";

export type NodeSection = {
  readonly id: string;
  readonly label: string;
  readonly icon: IconSvgElement;
  readonly to: Extract<RoutePath, `/nodes/$nodeId${string}`>;
  /** Whether a Node has the section; absent, every Node has it. */
  readonly shows?: (node: NodesNode) => boolean;
};

/** A Node that stands in the Role in any state: holding it, away, covering, draining or writing it off. */
const standsIn =
  (role: NodesRoleViewRole) =>
  (node: NodesNode): boolean =>
    node.roles.some((view) => view.role === role);

/**
 * A Node's sections, in order. They draw the desktop page tabs, the phone
 * Node page's rows and the palette's Go to commands. A later per-Node screen
 * joins as a row here.
 */
export const nodeSections = [
  { id: "overview", label: "Overview", icon: InformationCircleIcon, to: "/nodes/$nodeId" },
  { id: "roles", label: "Roles", icon: Layers01Icon, to: "/nodes/$nodeId/roles" },
  { id: "web", label: "Web and API", icon: Globe02Icon, to: "/nodes/$nodeId/web" },
  {
    id: "torrent-client",
    label: "Torrent client",
    icon: Download04Icon,
    to: "/nodes/$nodeId/torrent-client",
    shows: standsIn("downloading"),
  },
  { id: "logs", label: "Logs", icon: TextAlignLeftIcon, to: "/nodes/$nodeId/logs" },
  {
    id: "log-settings",
    label: "Log settings",
    icon: Settings02Icon,
    to: "/nodes/$nodeId/log-settings",
  },
] as const satisfies readonly NodeSection[];

export type NodeSectionId = (typeof nodeSections)[number]["id"];

/** The sections a Node has, in order. */
export const sectionsFor = (node: NodesNode): readonly NodeSection[] =>
  nodeSections.filter((section: NodeSection) => section.shows?.(node) ?? true);

export function sectionOf(id: NodeSectionId): NodeSection {
  const section = nodeSections.find((candidate) => candidate.id === id);
  if (section === undefined) {
    throw new Error(`no Node section ${id}`);
  }
  return section;
}

/** The palette's Go to command for each section of the Node the browser talks to. */
export function useNodeSectionCommands(): Command[] {
  const self = useQuery({
    queryKey: getListNodesQueryKey(),
    queryFn: ({ signal }) => listNodes({ signal }),
  }).data?.nodes.find((node) => node.self);
  if (self === undefined) {
    return [];
  }
  return sectionsFor(self).map((section) => ({
    id: `go-node-${section.id}`,
    label: `Go to ${section.label}`,
    meta: self.name,
    icon: section.icon,
    audience: "admin",
    run: ({ navigate }) => {
      void navigate({ to: section.to, params: { nodeId: self.id } });
    },
  }));
}
