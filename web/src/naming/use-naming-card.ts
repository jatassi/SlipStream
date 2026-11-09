import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useRouter } from "@tanstack/react-router";
import { useEffect, useState } from "react";

import { isProblem, sentenceOf } from "../api/errors";
import type { NamingPreview, NamingTypeNaming, WorksContentType } from "../api/gen/model";
import { useCountUnmatchedSuspense } from "../api/gen/query/files/files";
import { getPreviewNamingUrl, previewNaming, setNaming } from "../api/gen/query/naming/naming";
import type { Draft } from "./card";
import { recommend } from "./recommendation";
import { useRenamePlan } from "./save-flow";

const pause = 300;

/** The paths the type's samples get under the draft, read again once typing pauses. */
function usePreview(contentType: string, draft: Draft): NamingPreview | undefined {
  const [settled, setSettled] = useState(draft);
  useEffect(() => {
    const timer = setTimeout(() => {
      setSettled(draft);
    }, pause);
    return () => {
      clearTimeout(timer);
    };
  }, [draft]);
  const body = { draft: Object.fromEntries(settled) };
  return useQuery({
    queryKey: [getPreviewNamingUrl(contentType), body],
    queryFn: ({ signal }) => previewNaming(contentType, body, { signal }),
    placeholderData: keepPreviousData,
  }).data;
}

/** A refused save: each sentence with the key of the setting it is about, when it names one. */
export type Refusal = readonly { readonly key?: string; readonly sentence: string }[];

function refusalOf(error: unknown): Refusal {
  if (isProblem(error, "naming_template_refused")) {
    return error.errors.map(({ location, message }) => ({
      key: location.replace(/^body\.(draft\.)?values\./u, ""),
      sentence: message,
    }));
  }
  return [{ sentence: sentenceOf(error) }];
}

/**
 * A card's unsaved values, their preview, what it leads with, and the one
 * save every action on the card goes through. A save that renames files
 * first shows the rename plan for the draft.
 */
export function useNamingCard(type: WorksContentType, naming: NamingTypeNaming) {
  const router = useRouter();
  const [draft, setDraft] = useState<Draft>(new Map());
  const [refusal, setRefusal] = useState<Refusal>([]);
  const [busy, setBusy] = useState(false);
  const preview = usePreview(naming.contentType, draft);
  const unmatched = useCountUnmatchedSuspense().data.contentTypes.find(
    ({ contentType }) => contentType === type.id,
  );
  const plan = useRenamePlan(type, async (values) => {
    await setNaming(naming.contentType, { values });
    await router.invalidate();
    setDraft((current) => new Map([...current].filter(([key]) => !(key in values))));
  });

  const save = async (values: Readonly<Record<string, unknown>>) => {
    setBusy(true);
    setRefusal([]);
    try {
      await plan.save(values);
    } catch (error: unknown) {
      setRefusal(refusalOf(error));
    } finally {
      setBusy(false);
    }
  };

  return {
    draft,
    preview,
    refusal,
    busy,
    lead: recommend(type, naming, unmatched?.files ?? 0),
    review: plan.review,
    sheet: plan.sheet,
    change: (next: Draft) => {
      setDraft(next);
      setRefusal([]);
    },
    save,
    saveDraft: () => save(Object.fromEntries(draft)),
    discard: () => {
      setDraft(new Map());
      setRefusal([]);
    },
  };
}
