import { useListDownloadClientsSuspense } from "../api/gen/query/downloadclients/downloadclients";
import { useListNodesSuspense } from "../api/gen/query/nodes/nodes";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { type Row, builtInClient, clientOf } from "./model";

export type Choice = { readonly id: string; readonly name: string };

// The Nodes, Content Types and clients the Torrents fall under, plus the one chosen,
// so a link to a Node holding none still reads its name.
function choicesOf(
  rows: readonly Row[],
  of: (row: Row) => string,
  named: readonly Choice[],
  picked: string | undefined,
): Choice[] {
  const held = new Set(rows.map((row) => of(row)));
  return named.filter(({ id }) => held.has(id) || id === picked);
}

/** The Nodes, Content Types and clients the Torrents view can filter by. */
export function useChoices(
  rows: readonly Row[],
  {
    node,
    type,
    client,
  }: { node: string | undefined; type: string | undefined; client: string | undefined },
) {
  const { cards } = useListDownloadClientsSuspense().data;
  const clients = [
    { id: builtInClient, name: "Built-in client" },
    ...cards.flatMap((card) =>
      card.externals.map(({ id, name }) => ({ id, name: `${name} on ${card.nodeName}` })),
    ),
  ];
  return {
    clients: choicesOf(rows, clientOf, clients, client),
    nodes: choicesOf(rows, (row) => row.nodeId, useListNodesSuspense().data.nodes, node),
    types: choicesOf(
      rows,
      (row) => row.contentType,
      useListContentTypesSuspense().data.contentTypes,
      type,
    ),
  };
}
