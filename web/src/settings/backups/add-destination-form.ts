import { useState } from "react";

import { sentenceOf } from "../../api/errors";
import type { NodesNode } from "../../api/gen/model";
import { addBackupDestination } from "../../api/gen/query/backups/backups";
import { useAppForm } from "../../ui/form";
import { toast } from "../../ui/toast";
import { roomToast } from "./words";

export function useAddDestination(node: NodesNode | undefined, close: () => void) {
  const [refusal, setRefusal] = useState<string>();
  const form = useAppForm({
    defaultValues: { path: "" },
    onSubmit: async ({ value }) => {
      if (node === undefined) {
        return;
      }
      setRefusal(undefined);
      try {
        const { room } = await addBackupDestination({ nodeId: node.id, folder: value.path.trim() });
        toast(
          room === null
            ? `Added a Backup destination on ${node.name}. SlipStream copies every Backup there.`
            : roomToast(room.need, room.free, node.name),
        );
        close();
      } catch (error) {
        setRefusal(sentenceOf(error));
      }
    },
  });
  return { form, refusal };
}
