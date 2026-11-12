import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import { sentenceOf } from "../api/errors";
import type {
  ImportsFileAction,
  ImportsForceOption,
  ImportsImportEntry,
  ImportsImportFile,
} from "../api/gen/model";
import { actOnImportFile } from "../api/gen/query/imports/imports";
import { toast } from "../ui/toast";
import type { Choice } from "./action-handlers";

/** A dialog a choice opens before anything is sent. */
export type Open =
  | { readonly kind: "force"; readonly label: string; readonly option: ImportsForceOption }
  | { readonly kind: "match" };

/** Runs a file's choices: posts an action, opens the Force confirm or the match sheet, or goes to a page. */
export function useChoosing(entry: ImportsImportEntry, file: ImportsImportFile) {
  const navigate = useNavigate();
  const [open, setOpen] = useState<Open>();
  const [busy, setBusy] = useState(false);
  const send = (body: ImportsFileAction) => actOnImportFile(entry.id, file.position, body);
  const post = async (body: ImportsFileAction) => {
    setBusy(true);
    try {
      await send(body);
    } catch (error) {
      toast(sentenceOf(error), { kind: "fail" });
    } finally {
      setBusy(false);
    }
  };
  const choose = ({ label, step }: Choice) => {
    switch (step.kind) {
      case "post":
        void post(step.body);
        return;
      case "force":
        setOpen({ kind: "force", label, option: step.option });
        return;
      case "match":
        setOpen({ kind: "match" });
        return;
      case "editions":
        void navigate({ to: "/works/$workId", params: { workId: step.workId } });
        return;
      case "settings":
        void navigate({ to: "/settings/$section", params: { section: step.section } });
        return;
      default:
        step satisfies never;
    }
  };
  const close = () => {
    setOpen(undefined);
  };
  return { open, close, busy, choose, send };
}
