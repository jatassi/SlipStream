import { useState } from "react";

import { isProblem, sentenceOf } from "../api/errors";
import type { IndexersIndexer } from "../api/gen/model";
import { getIndexerCaptcha, testIndexer } from "../api/gen/query/indexers/indexers";
import { countWords } from "../format";
import { useAppForm } from "../ui/form";
import type { CaptchaStep } from "./add/use-add";
import { captchaOf } from "./captcha-of";

const items = { one: "item", many: "items" } as const;

export type TestResult = { readonly passed: boolean; readonly words: string };

/** Where the captcha stands after a failed Test, and what the page says. */
function afterFailure(error: unknown): { step: CaptchaStep; result?: TestResult } {
  if (isProblem(error, "captcha_expired")) {
    return { step: { kind: "expired", signIn: "", words: sentenceOf(error) } };
  }
  const captcha = captchaOf(error);
  return {
    step: captcha === undefined ? { kind: "none" } : { kind: "shown", captcha, signIn: "" },
    result: { passed: false, words: sentenceOf(error) },
  };
}

/** The Test's answer to the captcha step shows, when one does. */
const answerOf = (step: CaptchaStep, code: string) =>
  step.kind === "shown"
    ? { captcha: { attempt: step.captcha.attempt, answer: code.trim() } }
    : undefined;

/**
 * A saved Indexer's Test, with the code of the captcha its sign-in shows
 * once a Test hands one back. Its new status arrives with
 * indexers.status-changed.
 */
export function useIndexerTest(indexer: IndexersIndexer) {
  const [step, setStep] = useState<CaptchaStep>({ kind: "none" });
  const [result, setResult] = useState<TestResult>();
  const form = useAppForm({
    defaultValues: { captcha: "" },
    onSubmit: async ({ value, formApi }) => {
      setResult(undefined);
      try {
        const report = await testIndexer(indexer.id, answerOf(step, value.captcha));
        setStep({ kind: "none" });
        const listed = countWords(report.items, items);
        setResult({ passed: true, words: `${indexer.name} passed the Test and listed ${listed}.` });
      } catch (error) {
        formApi.setFieldValue("captcha", "");
        const after = afterFailure(error);
        setStep(after.step);
        setResult(after.result);
      }
    },
  });
  const reload = async () => {
    setResult(undefined);
    try {
      const captcha = await getIndexerCaptcha(indexer.id);
      form.setFieldValue("captcha", "");
      setStep({ kind: "shown", captcha, signIn: "" });
    } catch (error) {
      setResult({ passed: false, words: sentenceOf(error) });
    }
  };
  return { form, step, result, reload };
}

export type TestForm = ReturnType<typeof useIndexerTest>["form"];
