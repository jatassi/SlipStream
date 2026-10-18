import type { ReactNode } from "react";

import { type Part, selectedOf } from "../../../../diagram/parts-order";
import type { Acting } from "../-manage/dialogs";
import { LinkWords } from "./inspect-link";
import { NodeWords } from "./inspect-node";
import { RoleWords } from "./inspect-role";
import type { Landing } from "./view";

export type Words = { readonly title: string; readonly body: ReactNode };

/** The Inspector's title and body for a selected part, or undefined when the part is gone from the landing. */
export function wordsOf(
  landing: Landing,
  part: Part,
  act: (acting: Acting) => void,
): Words | undefined {
  const selected = selectedOf(part);
  if (selected.kind === "link") {
    const link = landing.links.find(({ worker }) => worker === selected.worker);
    const worker = landing.nodes.find(({ id }) => id === selected.worker);
    return link === undefined || worker === undefined
      ? undefined
      : {
          title: `${landing.controller} and ${worker.name}`,
          body: <LinkWords landing={landing} link={link} worker={worker} />,
        };
  }
  const node = landing.nodes.find(({ id }) => id === selected.node);
  if (node === undefined) {
    return undefined;
  }
  if (selected.kind === "node") {
    return { title: node.name, body: <NodeWords node={node} act={act} /> };
  }
  const chip = node.roles.find(({ role }) => role === selected.role);
  return chip === undefined
    ? undefined
    : { title: `${chip.label} on ${node.name}`, body: <RoleWords node={node} chip={chip} /> };
}
