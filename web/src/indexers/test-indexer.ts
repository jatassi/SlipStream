import { useState } from "react";

import { isProblem, sentenceOf } from "../api/errors";
import type { IndexersIndexer } from "../api/gen/model";
import { getIndexerCaptcha, testIndexer } from "../api/gen/query/indexers/indexers";
import { countWords } from "../format";
import { useAppForm } from "../ui/form";
import type { SignInStep } from "./add/use-add";
import { captchaOf, needsCode } from "./captcha-of";

const items = { one: "item", many: "items" } as const;

export type TestResult = { readonly passed: boolean; readonly words: string };

/** Where the sign-in step stands after a failed Test, and what the page says. */
function afterFailure(error: unknown): { step: SignInStep; result?: TestResult } {
  if (isProblem(error, "captcha_expired")) {
    return { step: { kind: "expired", signIn: "", words: sentenceOf(error) } };
  }
  const result = { passed: false, words: sentenceOf(error) };
  if (needsCode(error)) {
    return { step: { kind: "code" }, result };
  }
  const captcha = captchaOf(error);
  return {
    step: captcha === undefined ? { kind: "none" } : { kind: "shown", captcha, signIn: "" },
    result,
  };
}

type Answers = { captcha: string; code: string };

/** The Test's answer to the step that shows, when one does. */
function answerOf(step: SignInStep, { captcha, code }: Answers) {
  if (step.kind === "shown") {
    return { captcha: { attempt: step.captcha.attempt, answer: captcha.trim() } };
  }
  return step.kind === "code" ? { code: code.trim() } : undefined;
}

/**
 * A saved Indexer's Test, with the code of the captcha its sign-in shows
 * once a Test hands one back, or the two-factor code a sign-in needs. The
 * code step opens at the start when the page is opened for it, as Health's
 * Type code does. Its new status arrives with indexers.status-changed.
 */
export function useIndexerTest(indexer: IndexersIndexer, codeAsked: boolean) {
  const [step, setStep] = useState<SignInStep>(
    codeAsked && indexer.status.cause === "code_needed" ? { kind: "code" } : { kind: "none" },
  );
  const [result, setResult] = useState<TestResult>();
  const form = useAppForm({
    defaultValues: { captcha: "", code: "" } satisfies Answers,
    onSubmit: async ({ value, formApi }) => {
      setResult(undefined);
      try {
        const report = await testIndexer(indexer.id, answerOf(step, value));
        setStep({ kind: "none" });
        const listed = countWords(report.items, items);
        setResult({ passed: true, words: `${indexer.name} passed the Test and listed ${listed}.` });
      } catch (error) {
        formApi.setFieldValue("captcha", "");
        formApi.setFieldValue("code", "");
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
  const askCode = () => {
    setResult(undefined);
    setStep({ kind: "code" });
  };
  return { form, step, result, reload, askCode };
}

export type TestForm = ReturnType<typeof useIndexerTest>["form"];
