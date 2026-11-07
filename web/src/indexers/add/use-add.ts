import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";

import { fieldErrors, isApiError, isProblem, sentenceOf } from "../../api/errors";
import type {
  IndexersCaptchaImage,
  IndexersNewIndexer,
  IndexersReport,
  NodesNode,
} from "../../api/gen/model";
import {
  createIndexer,
  getIndexerCaptchaDraft,
  testIndexerDraft,
} from "../../api/gen/query/indexers/indexers";
import { useAppForm } from "../../ui/form";
import { captchaOf } from "../captcha-of";
import { type SubmitErrors, submitErrors } from "../../ui/form-context";
import { toast } from "../../ui/toast";
import {
  type AddValues,
  type Source,
  draftOf,
  feedOf,
  fieldOf,
  signInOf,
  startValues,
} from "./source";

/**
 * Where a captcha definition's sign-in stands: no captcha yet, one shown for
 * a username on a Node, or one that expired before it was answered.
 */
export type CaptchaStep =
  | { readonly kind: "none" }
  | { readonly kind: "shown"; readonly captcha: IndexersCaptchaImage; readonly signIn: string }
  | { readonly kind: "expired"; readonly signIn: string; readonly words: string };

/** A Torznab feed's passing Test, for the feed it read. */
export type Tested = { readonly feed: string; readonly report: IndexersReport };

export type Next = "test" | "captcha" | "save";

/** What the primary button does next for values. */
export function nextOf(
  source: Source,
  values: AddValues,
  step: CaptchaStep,
  tried: Tested | undefined,
): Next {
  if (source.kind === "torznab" && tried?.feed !== feedOf(values)) {
    return "test";
  }
  if (
    source.kind === "catalog" &&
    source.form.captcha &&
    !(step.kind === "shown" && step.signIn === signInOf(source, values))
  ) {
    return "captcha";
  }
  return "save";
}

const tested = new Set(["indexer_test_failed", "indexer_unavailable"]);

/** A refusal as the form shows it: a Test's sentence whole, a field's message beside it. */
function refusalOf(source: Source, error: unknown): SubmitErrors {
  if (isApiError(error) && tested.has(error.code)) {
    return { form: sentenceOf(error), fields: {} };
  }
  if (isApiError(error) && error.code === "invalid") {
    return {
      fields: Object.fromEntries(
        Object.entries(fieldErrors(error)).map(([at, message]) => [fieldOf(source, at), message]),
      ),
    };
  }
  return submitErrors(error, { indexer_name_taken: "name" });
}

type Steps = {
  source: Source;
  setStep: (step: CaptchaStep) => void;
  setTried: (tried: Tested) => void;
  setField: (name: "captcha" | "trackerName", value: string) => void;
};

/** Tests a Torznab feed, naming its Tracker after the Indexer when its items name none. */
async function test(steps: Steps, draft: IndexersNewIndexer, value: AddValues) {
  const report = await testIndexerDraft(draft);
  steps.setTried({ feed: feedOf(value), report });
  if (report.sources.length === 0 && value.trackerName.trim() === "") {
    steps.setField("trackerName", value.name.trim());
  }
}

async function loadCaptcha(steps: Steps, draft: IndexersNewIndexer, value: AddValues) {
  const captcha = await getIndexerCaptchaDraft(draft);
  steps.setField("captcha", "");
  steps.setStep({ kind: "shown", captcha, signIn: signInOf(steps.source, value) });
}

/** Saves the draft, with the answer to the captcha the step shows. */
function save(draft: IndexersNewIndexer, step: CaptchaStep, value: AddValues) {
  return createIndexer(
    step.kind === "shown"
      ? { ...draft, captcha: { attempt: step.captcha.attempt, answer: value.captcha.trim() } }
      : draft,
  );
}

/** After a refused save: a new captcha when the Test handed one back, else none until Continue. */
function afterRefusal(steps: Steps, error: unknown, value: AddValues) {
  const signIn = signInOf(steps.source, value);
  if (isProblem(error, "captcha_expired")) {
    steps.setStep({ kind: "expired", signIn, words: sentenceOf(error) });
    return;
  }
  const captcha = captchaOf(error);
  steps.setField("captcha", "");
  steps.setStep(captcha === undefined ? { kind: "none" } : { kind: "shown", captcha, signIn });
}

/**
 * The add form and its steps. The primary button tests a Torznab feed
 * first, so its report can name the Tracker; loads a captcha before a
 * captcha definition saves; and otherwise saves, which tests on the home
 * Node first. A saved Indexer opens its page.
 */
export function useAdd(source: Source, home: NodesNode | undefined, homeShown: boolean) {
  const [step, setStep] = useState<CaptchaStep>({ kind: "none" });
  const [tried, setTried] = useState<Tested>();
  const navigate = useNavigate();
  const form = useAppForm({
    defaultValues: startValues(source, home),
    onSubmit: async ({ value, formApi }) => {
      const steps: Steps = { source, setStep, setTried, setField: formApi.setFieldValue };
      const draft = draftOf(source, value, homeShown);
      const next = nextOf(source, value, step, tried);
      try {
        if (next === "test") {
          await test(steps, draft, value);
          return;
        }
        if (next === "captcha") {
          await loadCaptcha(steps, draft, value);
          return;
        }
        const made = await save(draft, step, value);
        toast(`Added ${made.name}`);
        await navigate({ to: "/settings/indexers/$indexerId", params: { indexerId: made.id } });
      } catch (error) {
        if (next === "save") {
          afterRefusal(steps, error, value);
        }
        if (!isProblem(error, "captcha_expired")) {
          formApi.setErrorMap({ onSubmit: refusalOf(source, error) });
        }
      }
    },
  });
  const reload = async () => {
    const { values } = form.state;
    form.setErrorMap({ onSubmit: undefined });
    const steps: Steps = { source, setStep, setTried, setField: form.setFieldValue };
    try {
      await loadCaptcha(steps, draftOf(source, values, homeShown), values);
    } catch (error) {
      form.setErrorMap({ onSubmit: refusalOf(source, error) });
    }
  };
  return { form, step, tried, reload };
}

export type AddForm = ReturnType<typeof useAdd>["form"];
