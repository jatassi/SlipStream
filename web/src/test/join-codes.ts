import { HttpResponse, http } from "msw";

import type { NodesCodeState, NodesJoinCode, NodesNode } from "../api/gen/model";
import { DialInputBody } from "../api/gen/zod/slipStream.schemas.zod";
import { problem } from "./handlers";

type Code = NodesJoinCode & { createdAt: string } & NodesCodeState;

const ids = [
  "1a2b3c4d-5e6f-4a7b-8c9d-0e1f2a3b4c5d",
  "2b3c4d5e-6f7a-4b8c-9d0e-1f2a3b4c5d6e",
  "3c4d5e6f-7a8b-4c9d-8e0f-2a3b4c5d6e7f",
];

export const codeText = (n: number) =>
  `SLP1-${String(n).repeat(5)}-ABCDE-FGHJK-MNPQR-STVWX-YZ012-34567-89ABC-DEFGH-JKMNP-QRSTV`;

type Store = {
  readonly codes: Map<string, Code>;
  readonly dialed: string[];
  readonly mint: () => Code;
};

const stateOf = ({ state, nodeId, dialing }: Code): NodesCodeState => ({
  state,
  ...(nodeId !== undefined && { nodeId }),
  ...(dialing !== undefined && { dialing }),
});

const notFound = () => problem(404, "join_code_not_found", "No Join Code has this id");

function dialHandler({ codes, dialed }: Store) {
  return http.post("*/api/v1/join-codes/:id/dial", async ({ params, request }) => {
    const { nodeAddress } = DialInputBody.parse(await request.json());
    const code = codes.get(String(params["id"]));
    if (!nodeAddress.startsWith("SLN1-")) {
      return problem(
        422,
        "node_address_damaged",
        "The Node address is mistyped, cut short or not a Node address",
      );
    }
    if (code === undefined) {
      return notFound();
    }
    dialed.push(nodeAddress);
    code.dialing = {
      address: nodeAddress,
      addresses: ["203.0.113.9:8485"],
      since: new Date().toISOString(),
      unreachable: false,
    };
    return new HttpResponse(null, { status: 202 });
  });
}

function handlersOf(store: Store) {
  const { codes, mint } = store;
  return [
    http.post("*/api/v1/join-codes", () => {
      const { id, code, expiresAt } = mint();
      return HttpResponse.json({ id, code, expiresAt }, { status: 201 });
    }),
    http.get("*/api/v1/join-codes", () =>
      HttpResponse.json({
        codes: [...codes.values()]
          .filter(({ state }) => state === "open")
          .map(({ id, createdAt, expiresAt }) => ({ id, createdAt, expiresAt })),
      }),
    ),
    http.get("*/api/v1/join-codes/:id", ({ params }) => {
      const code = codes.get(String(params["id"]));
      return code === undefined ? notFound() : HttpResponse.json(stateOf(code));
    }),
    http.delete("*/api/v1/join-codes/:id", ({ params }) => {
      codes.delete(String(params["id"]));
      return new HttpResponse(null, { status: 204 });
    }),
    dialHandler(store),
  ];
}

function minter(codes: Map<string, Code>, lifetimeMs: number) {
  let made = 0;
  return (): Code => {
    const id = ids[made];
    if (id === undefined) {
      throw new Error("the join code server ran out of ids");
    }
    made += 1;
    const now = Date.now();
    const code: Code = {
      id,
      code: codeText(made),
      createdAt: new Date(now).toISOString(),
      expiresAt: new Date(now + lifetimeMs).toISOString(),
      state: "open",
    };
    codes.set(id, code);
    return code;
  };
}

/**
 * The Join Code operations over codes held in memory, which a test then
 * uses or lets the dial fail. Codes last lifetimeMs from when they are made.
 */
export function joinCodeServer({ lifetimeMs = 60 * 60 * 1000 } = {}) {
  const codes = new Map<string, Code>();
  const store: Store = { codes, dialed: [], mint: minter(codes, lifetimeMs) };
  const find = (id: string): Code => {
    const code = codes.get(id);
    if (code === undefined) {
      throw new Error(`the join code server has no code ${id}`);
    }
    return code;
  };
  return {
    handlers: handlersOf(store),
    dialed: store.dialed,
    mint: store.mint,
    /** The Pair Write: node paired with the code. */
    use(id: string, node: NodesNode) {
      const code = find(id);
      code.state = "used";
      code.nodeId = node.id;
      delete code.dialing;
    },
    /** dial_report passed with no Pair. */
    unreachable(id: string) {
      const { dialing } = find(id);
      if (dialing === undefined) {
        throw new Error(`code ${id} has no dial`);
      }
      dialing.unreachable = true;
    },
  };
}

/** listNodes answering whatever nodes holds when asked. */
export const nodesFrom = (nodes: () => readonly NodesNode[]) =>
  http.get("*/api/v1/nodes", () => HttpResponse.json({ nodes: nodes() }));
