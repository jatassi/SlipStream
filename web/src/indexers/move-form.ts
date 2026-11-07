import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import type { IndexersIndexer, NodesNode } from "../api/gen/model";
import { updateIndexer } from "../api/gen/query/indexers/indexers";
import { adoptIndexer } from "../live/adopt";
import { useAppForm } from "../ui/form";
import { toast } from "../ui/toast";
import type { CaptchaStep } from "./add/use-add";
import { type MoveValues, moveOf, refusedMove, stepAfter } from "./move-edit";
import { nodeName } from "./words";

/**
 * The Move dialog's form. A move tests the Indexer from its new home first,
 * and a Test that meets a captcha there hands one back for the next Move
 * to answer.
 */
export function useMoveForm(
  indexer: IndexersIndexer,
  nodes: readonly NodesNode[],
  onDone: () => void,
) {
  const queryClient = useQueryClient();
  const [step, setStep] = useState<CaptchaStep>({ kind: "none" });
  const values: MoveValues = { to: indexer.homeNodeId, cookie: "", userAgent: "", captcha: "" };
  const form = useAppForm({
    defaultValues: values,
    onSubmit: async ({ value, formApi }) => {
      try {
        adoptIndexer(queryClient, await updateIndexer(indexer.id, moveOf(indexer, value, step)));
      } catch (error) {
        formApi.setFieldValue("captcha", "");
        setStep(stepAfter(error, value.to));
        formApi.setErrorMap({ onSubmit: refusedMove(error) });
        return;
      }
      toast(`${indexer.name} searches from ${nodeName(nodes, value.to)} now`);
      onDone();
    },
  });
  return { form, step };
}

export type MoveForm = ReturnType<typeof useMoveForm>["form"];
