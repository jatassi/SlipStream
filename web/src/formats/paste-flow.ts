import { useState } from "react";

import { fieldErrors, isApiError, isProblem, sentenceOf } from "../api/errors";
import type { FormatsPasted } from "../api/gen/model";
import { pasteCustomFormats } from "../api/gen/query/formats/formats";
import type { ImpactChange } from "../targets/use-impact-save";
import { toast } from "../ui/toast";
import { type App, pasteLabel, replacedAtFirst } from "./paste-words";

export type Previewed = {
  readonly step: "preview";
  readonly app: App;
  readonly replace: readonly string[];
  readonly pasted: FormatsPasted;
};

/**
 * Where a paste is: the text alone, the text with the app to ask for
 * because its conditions don't settle it, or a preview of what saving adds
 * and replaces.
 */
export type Step = { readonly step: "text" } | { readonly step: "app" } | Previewed;

export const preview = (text: string, app: App | undefined, replace: readonly string[]) =>
  pasteCustomFormats(
    { text, ...(app !== undefined && { app }), replace: [...replace] },
    { preview: true },
  );

export async function firstPreview(text: string, app: App | undefined): Promise<Previewed> {
  const plain = await preview(text, app, []);
  const replace = replacedAtFirst(plain);
  const pasted = replace.length === 0 ? plain : await preview(text, plain.app, replace);
  return { step: "preview", app: pasted.app, replace, pasted };
}

export const withReplace = async (
  text: string,
  shown: Previewed,
  replace: readonly string[],
): Promise<Previewed> => ({ ...shown, replace, pasted: await preview(text, shown.app, replace) });

export const savingOf = (text: string, shown: Previewed, onDone: () => void): ImpactChange => ({
  title: "Paste Custom Formats",
  action: pasteLabel(shown.pasted.formats),
  run: (changing) =>
    pasteCustomFormats({ text, app: shown.app, replace: [...shown.replace] }, changing),
  done: () => {
    const count = shown.pasted.formats.length;
    toast(count === 1 ? "Pasted 1 Custom Format." : `Pasted ${String(count)} Custom Formats.`);
    onDone();
  },
});

const problemOf = (error: unknown): string =>
  (isApiError(error) ? fieldErrors(error)["text"] : undefined) ?? sentenceOf(error);

export function usePasteSteps() {
  const [text, setText] = useState("");
  const [step, setStep] = useState<Step>({ step: "text" });
  const [problem, setProblem] = useState<string>();
  const [busy, setBusy] = useState(false);
  const attempt = async (next: () => Promise<Step>) => {
    setBusy(true);
    setProblem(undefined);
    try {
      setStep(await next());
    } catch (error) {
      if (isProblem(error, "format_app_needed")) {
        setStep({ step: "app" });
      } else {
        setProblem(problemOf(error));
      }
    } finally {
      setBusy(false);
    }
  };
  return {
    text,
    step,
    problem,
    busy,
    attempt: (next: () => Promise<Step>) => void attempt(next),
    edit: (next: string) => {
      setText(next);
      setStep({ step: "text" });
    },
  };
}
