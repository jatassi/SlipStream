import { isProblem, sentenceOf } from "../api/errors";
import { useMakeFolder } from "../api/gen/query/roots/roots";
import type { BrowserAction, NewFolder } from "./browser-state";

function refusalOf(error: unknown, name: string, nodeName: string): string {
  if (isProblem(error, "folder_exists")) {
    return `A folder named ${name} is already here.`;
  }
  if (isProblem(error, "folder_unreadable")) {
    return `SlipStream on ${nodeName} can't make a folder here.`;
  }
  return sentenceOf(error);
}

/** Makes the New folder row's folder in `parent`, then opens it; a refusal stays in the row. */
export function useNewFolder({
  nodeId,
  nodeName,
  parent,
  creating,
  dispatch,
  open,
}: {
  nodeId: string;
  nodeName: string;
  parent: string | undefined;
  creating: NewFolder | null;
  dispatch: (action: BrowserAction) => void;
  open: (path: string) => void;
}) {
  const making = useMakeFolder();
  const create = () => {
    const name = creating?.name.trim() ?? "";
    if (parent === undefined || name === "" || making.isPending) {
      return;
    }
    making.mutate(
      { id: nodeId, data: { parent, name } },
      {
        onSuccess: (made) => {
          open(made.path);
        },
        onError: (error) => {
          dispatch({ type: "folderRefused", error: refusalOf(error, name, nodeName) });
        },
      },
    );
  };
  return { busy: making.isPending, create };
}
