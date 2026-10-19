import type { RootsRoot } from "../api/gen/model";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { Dialog } from "../ui/dialog";
import { AddRootForm } from "./add-root";
import { RehomeForm } from "./rehome";
import { typeLabel } from "./words";

/** Add a library root, or edit one's place (its Content Type stays). */
export function LibraryRootDialog({
  open,
  onOpenChange,
  root,
  contentType,
  onSaved,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  root?: RootsRoot | undefined;
  contentType?: string | undefined;
  onSaved?: (root: RootsRoot) => void;
}) {
  const { contentTypes } = useListContentTypesSuspense().data;
  const close = () => {
    onOpenChange(false);
  };
  const saved = (made: RootsRoot) => {
    onSaved?.(made);
    close();
  };
  return (
    <Dialog
      title={
        root === undefined
          ? "Add a library root"
          : `Edit the ${typeLabel(contentTypes, root.contentType)} root`
      }
      open={open}
      onOpenChange={onOpenChange}
    >
      {root === undefined ? (
        <AddRootForm
          contentType={contentType ?? contentTypes.find((type) => type.enabled)?.id ?? ""}
          onSaved={saved}
          onCancel={close}
        />
      ) : (
        <RehomeForm root={root} onSaved={saved} onCancel={close} />
      )}
    </Dialog>
  );
}
