import { useState } from "react";

import type { RootsRoot } from "../api/gen/model";
import { deleteRoot, useListRootsSuspense } from "../api/gen/query/roots/roots";
import { useListContentTypesSuspense } from "../api/gen/query/works/works";
import { type Noun, countWords } from "../format";
import { ConfirmDialog } from "../ui/confirm-dialog";
import { Select } from "../ui/select";
import { toast } from "../ui/toast";
import { KeepDefaultDialog } from "./keep-default";
import { deleteRefusal } from "./refused";

type DialogProps = {
  root: RootsRoot;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

function WorksGo({
  root,
  work,
  others,
  worksTo,
  onWorksToChange,
}: {
  root: RootsRoot;
  work: Noun;
  others: readonly RootsRoot[];
  worksTo: string | undefined;
  onWorksToChange: (worksTo: string) => void;
}) {
  if (root.works === 0) {
    return null;
  }
  const works = countWords(root.works, work);
  if (worksTo === undefined) {
    return (
      <p>
        Its {works} wait without a root until you add another {work.many} root.
      </p>
    );
  }
  return (
    <>
      <p>Its {works} move to the root you pick.</p>
      <Select
        label={`Move its ${work.many} to`}
        options={others.map((candidate) => ({ value: candidate.id, label: candidate.path }))}
        value={worksTo}
        onValueChange={onWorksToChange}
      />
    </>
  );
}

/** Forget a root that holds no files; its folder stays on disk and its Works go to another root. */
export function DeleteRootDialog({ root, open, onOpenChange }: DialogProps) {
  const others = useListRootsSuspense().data.roots.filter(
    (candidate) => candidate.contentType === root.contentType && candidate.id !== root.id,
  );
  const { contentTypes } = useListContentTypesSuspense().data;
  const work = contentTypes.find((candidate) => candidate.id === root.contentType)?.work ?? {
    one: "Work",
    many: "Works",
  };
  const [worksTo, setWorksTo] = useState(
    (others.find((candidate) => candidate.default) ?? others[0])?.id,
  );
  if (root.default && others.length > 0) {
    return (
      <KeepDefaultDialog root={root} label={work.many} open={open} onOpenChange={onOpenChange} />
    );
  }
  const confirm = async () => {
    try {
      await deleteRoot(root.id, root.works > 0 && worksTo !== undefined ? { worksTo } : {});
    } catch (error) {
      throw new Error(deleteRefusal(error), { cause: error });
    }
    toast(`Deleted the ${root.path} root. The folder is still on disk.`);
  };
  return (
    <ConfirmDialog
      title={`Delete ${root.path}?`}
      confirmLabel="Delete"
      look="fail"
      open={open}
      onOpenChange={onOpenChange}
      onConfirm={confirm}
    >
      <p>SlipStream forgets this root. The folder and everything in it stay on disk.</p>
      <WorksGo
        root={root}
        work={work}
        others={others}
        worksTo={worksTo}
        onWorksToChange={setWorksTo}
      />
    </ConfirmDialog>
  );
}
