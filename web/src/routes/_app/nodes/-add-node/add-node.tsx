import { Add01Icon } from "@hugeicons/core-free-icons";
import { useNavigate, useSearch } from "@tanstack/react-router";
import { useState } from "react";

import { isProblem, sentenceOf } from "../../../../api/errors";
import type { NodesJoinCode } from "../../../../api/gen/model";
import { createJoinCode, useListNodesSuspense } from "../../../../api/gen/query/nodes/nodes";
import { Button } from "../../../../ui/button";
import { useRefreshCodes } from "./codes";
import { AddNodeDialog } from "./dialog";

const noAddress = (controller: string) =>
  `${controller} has no link address to put in a Join Code. Set link.advertise in its config.yaml, then restart it.`;

/** Mints a Join Code and follows it in the address, or says why none could be made. */
function useMint(controller: string) {
  const navigate = useNavigate();
  const refreshCodes = useRefreshCodes();
  const [minted, setMinted] = useState<NodesJoinCode>();
  const [minting, setMinting] = useState(false);
  const [failure, setFailure] = useState<string>();
  const follow = (id: string | undefined, replace: boolean) =>
    navigate({ to: ".", search: (prev) => ({ ...prev, add: id }), replace });
  const mint = async (replace: boolean) => {
    setMinting(true);
    setFailure(undefined);
    try {
      const code = await createJoinCode();
      setMinted(code);
      await follow(code.id, replace);
      await refreshCodes();
    } catch (error) {
      setFailure(isProblem(error, "no_link_address") ? noAddress(controller) : sentenceOf(error));
    } finally {
      setMinting(false);
    }
  };
  const close = () => {
    setFailure(undefined);
    void follow(undefined, true);
  };
  return { minted, minting, failure, mint, close };
}

/**
 * The Add a Node button and its dialog. The dialog follows the Join Code in
 * the address, so a reload keeps it open; the code's text lives only here.
 */
export function AddNode() {
  const { add } = useSearch({ from: "/_app/nodes" });
  const navigate = useNavigate();
  const controller =
    useListNodesSuspense().data.nodes.find(({ self }) => self)?.name ?? "The Controller";
  const { minted, minting, failure, mint, close } = useMint(controller);
  return (
    <>
      <Button icon={Add01Icon} busy={minting && add === undefined} onClick={() => void mint(false)}>
        Add a Node
      </Button>
      {(add !== undefined || failure !== undefined) && (
        <AddNodeDialog
          id={add}
          minted={minted}
          mint={{ minting, onMint: () => void mint(true) }}
          failure={failure}
          onClose={close}
          onAdded={(node) => {
            void navigate({ to: "/nodes/$nodeId/roles", params: { nodeId: node.id }, search: {} });
          }}
        />
      )}
    </>
  );
}
