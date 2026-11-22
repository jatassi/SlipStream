import { useState } from "react";

import { isApiError, sentenceOf } from "../api/errors";
import type { ConfirmRestoreInputBody, RestoreSummary } from "../api/gen/model";
import { getStatus } from "../api/gen/query/system/system";

export type RestoreCalls = {
  readonly upload: (file: File, onProgress: (sent: number) => void) => Promise<{ id: string }>;
  readonly open: (id: string, passphrase: string) => Promise<RestoreSummary>;
  readonly confirm: (id: string, body: ConfirmRestoreInputBody) => Promise<unknown>;
  /** Takes a refusal that leaves the flow, such as a Setup code from before a restart. */
  readonly leaves?: (error: unknown) => boolean;
};

/**
 * Where a restore stands. An upload names the staged file, the passphrase
 * opens it into a summary, and the confirm hands over to the restart, which
 * waits for a process that started after `before`.
 */
export type RestoreStep =
  | { readonly at: "file"; readonly refusal?: string | undefined }
  | { readonly at: "uploading"; readonly name: string; readonly sent: number }
  | { readonly at: "passphrase"; readonly id: string; readonly refusal?: string | undefined }
  | {
      readonly at: "summary";
      readonly id: string;
      readonly summary: RestoreSummary;
      readonly refusal?: string | undefined;
    }
  | { readonly at: "restarting"; readonly before: string };

// A wrong passphrase is retyped on the same upload. Every other refusal of
// the open means another file.
const retypes = (error: unknown) => isApiError(error) && error.code === "backup_wrong_passphrase";

const chooseFile: RestoreStep = { at: "file" };

export function useRestoreFlow(calls: RestoreCalls) {
  const [step, setStep] = useState<RestoreStep>(chooseFile);
  const refused = (error: unknown): string | undefined =>
    calls.leaves?.(error) === true ? undefined : sentenceOf(error);

  const upload = async (file: File) => {
    setStep({ at: "uploading", name: file.name, sent: 0 });
    try {
      const { id } = await calls.upload(file, (sent) => {
        setStep({ at: "uploading", name: file.name, sent });
      });
      setStep({ at: "passphrase", id });
    } catch (error) {
      setStep({ at: "file", refusal: refused(error) });
    }
  };

  const open = async (id: string, passphrase: string) => {
    try {
      setStep({ at: "summary", id, summary: await calls.open(id, passphrase) });
    } catch (error) {
      const refusal = refused(error);
      setStep(retypes(error) ? { at: "passphrase", id, refusal } : { at: "file", refusal });
    }
  };

  const confirm = async (
    current: Extract<RestoreStep, { at: "summary" }>,
    body: ConfirmRestoreInputBody,
  ) => {
    try {
      const { startedAt } = await getStatus();
      await calls.confirm(current.id, body);
      setStep({ at: "restarting", before: startedAt });
    } catch (error) {
      setStep({ ...current, refusal: refused(error) });
    }
  };

  const staged = (id: string, summary: RestoreSummary) => {
    setStep({ at: "summary", id, summary });
  };

  const reset = () => {
    setStep(chooseFile);
  };

  return { step, upload, open, confirm, staged, reset };
}

export type RestoreFlow = ReturnType<typeof useRestoreFlow>;
