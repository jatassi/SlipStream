import { useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import type {
  DownloadclientsExternalClient as External,
  DownloadclientsKind as Kind,
} from "../api/gen/model";
import {
  createDownloadClient,
  testDownloadClient,
  testDownloadClientDraft,
  updateDownloadClient,
} from "../api/gen/query/downloadclients/downloadclients";
import { adoptExternalClient } from "../live/adopt";
import { useAppForm } from "../ui/form";
import { toast } from "../ui/toast";
import {
  type ClientValues,
  type TestResult,
  answerOf,
  draftOf,
  editOf,
  kindOf,
  namesDetails,
  newValues,
  valuesOf,
} from "./client-form";

/** What the sheet opened on: a new client for a Node, or a saved client. */
export type Opened =
  | { readonly kind: "add"; readonly nodeId: string }
  | { readonly kind: "edit"; readonly client: External };

/** Saves the draft, answering the client as saved, or null when an edit changed nothing. */
async function save(saved: External | undefined, value: ClientValues, kind: Kind) {
  if (saved === undefined) {
    const made = await createDownloadClient(draftOf(value, kind));
    toast(`Added ${made.name}`);
    return made;
  }
  const edit = editOf(saved, value, kind);
  if (Object.keys(edit).length === 0) {
    return null;
  }
  const got = await updateDownloadClient(saved.id, edit);
  toast(`Saved ${got.name}`);
  return got;
}

const testOf = (saved: External | undefined, value: ClientValues, kind: Kind) =>
  saved === undefined
    ? testDownloadClientDraft(draftOf(value, kind))
    : testDownloadClient(saved.id, editOf(saved, value, kind));

/**
 * The client sheet's form. Test runs the draft on its Node, a saved client's
 * with its saved secrets. Save creates or edits, which tests on the Node
 * first. A failed Test shows its sentence and marks the fields it names,
 * opening Details when one sits there.
 */
export function useClientSheet(opened: Opened, kinds: readonly Kind[], onDone: () => void) {
  const queryClient = useQueryClient();
  const [result, setResult] = useState<TestResult>();
  const [testing, setTesting] = useState(false);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const saved = opened.kind === "edit" ? opened.client : undefined;
  const [first] = kinds;
  if (first === undefined) {
    throw new Error("the Listing has no kinds");
  }
  const form = useAppForm({
    defaultValues:
      opened.kind === "add" ? newValues(opened.nodeId, first) : valuesOf(opened.client),
    onSubmit: async ({ value }) => {
      setResult(undefined);
      try {
        const got = await save(saved, value, kindOf(kinds, value.kind));
        if (got !== null) {
          adoptExternalClient(queryClient, got);
        }
      } catch (error) {
        refused(error);
        return;
      }
      onDone();
    },
  });
  function refused(error: unknown) {
    const answer = answerOf(error);
    setResult(answer.result);
    form.setErrorMap({ onSubmit: answer.errors });
    if (namesDetails(answer.errors)) {
      setDetailsOpen(true);
    }
  }
  const test = async () => {
    const value = form.state.values;
    setTesting(true);
    setResult(undefined);
    form.setErrorMap({ onSubmit: { fields: {} } });
    try {
      setResult({ passed: true, report: await testOf(saved, value, kindOf(kinds, value.kind)) });
    } catch (error) {
      refused(error);
    } finally {
      setTesting(false);
    }
  };
  return { form, saved, result, testing, test, detailsOpen, setDetailsOpen };
}

export type ClientSheetState = ReturnType<typeof useClientSheet>;
export type ClientForm = ClientSheetState["form"];
