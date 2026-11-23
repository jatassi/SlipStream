import type { RootsRoot } from "../api/gen/model";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { RootMoveDialog } from "../moves/move-dialog";
import { Button } from "../ui/button";
import { Dialog } from "../ui/dialog";
import type { Acting } from "./acting-dialogs";

/**
 * "Move root…": leads with editing the root, which copies nothing, and
 * offers to move its Works to another root instead.
 */
export function MoveRootChooser({
  root,
  onAct,
  onOpenChange,
}: {
  root: RootsRoot;
  onAct: (acting: Acting) => void;
  onOpenChange: (open: boolean) => void;
}) {
  const many = useWorksNoun(root);
  return (
    <Dialog title={`Move ${root.path}`} open onOpenChange={onOpenChange}>
      <div className="flex flex-col gap-4">
        <p>
          If you moved this disk to another machine, edit the root. Nothing is copied. To copy its{" "}
          {many} to another root, move them.
        </p>
        <div className="flex flex-wrap justify-end gap-2">
          <Button
            onClick={() => {
              onAct({ kind: "move-works", rootId: root.id });
            }}
          >
            Move its {many}
          </Button>
          <Button
            look="primary"
            onClick={() => {
              onAct({ kind: "edit", rootId: root.id });
            }}
          >
            Edit the root
          </Button>
        </div>
      </div>
    </Dialog>
  );
}

const useType = (root: RootsRoot) =>
  useListContentTypesSuspense().data.contentTypes.find(({ id }) => id === root.contentType);

const useWorksNoun = (root: RootsRoot) => useType(root)?.work.many.toLowerCase() ?? "Works";

/** The Move Dialog for every Work of root. */
export function MoveRootWorks({
  root,
  onOpenChange,
}: {
  root: RootsRoot;
  onOpenChange: (open: boolean) => void;
}) {
  const type = useType(root);
  return type === undefined ? null : (
    <RootMoveDialog type={type} from={root} open onOpenChange={onOpenChange} />
  );
}
