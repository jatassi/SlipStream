import { z } from "zod";

import { DialogActions } from "../account/dialog-actions";
import type { SubtitlesProvider } from "../api/gen/model";
import { type KeyAnswer, useSubtitleWrites } from "../subtitles/settings-api";
import { keySteps, passwordWords } from "../subtitles/settings-words";
import { Button } from "../ui/button";
import { DialogClose } from "../ui/dialog";
import { useAppForm } from "../ui/form";
import { GuidedSteps } from "../ui/guided-steps";

const keyForm = z.object({ key: z.string().trim().min(1, "Paste the key.").max(256) });

function useKeyForm(provider: SubtitlesProvider, onAnswer: (answer: KeyAnswer) => void) {
  const { checkKey } = useSubtitleWrites();
  return useAppForm({
    defaultValues: { key: "" },
    validators: { onSubmit: keyForm },
    onSubmit: async ({ value }) => {
      onAnswer(await checkKey(provider.id, keyForm.parse(value).key));
    },
  });
}

type KeyForm = ReturnType<typeof useKeyForm>;

function KeyField({ form }: { form: KeyForm }) {
  return (
    <form.AppField name="key">
      {(field) => (
        <field.TextField
          label="Key"
          data
          autoComplete="off"
          spellCheck={false}
          onPaste={(event) => {
            const pasted = event.clipboardData.getData("text");
            if (pasted.trim() !== "") {
              event.preventDefault();
              field.handleChange(pasted);
              void form.handleSubmit();
            }
          }}
        />
      )}
    </form.AppField>
  );
}

/** The three steps, the key field and its check. */
export function KeySteps({
  provider,
  onAnswer,
}: {
  provider: SubtitlesProvider;
  onAnswer: (answer: KeyAnswer) => void;
}) {
  const form = useKeyForm(provider, onAnswer);
  const [open, copy, paste] = keySteps(provider);
  const link =
    provider.keyPage === null
      ? {}
      : { link: { label: `Open ${provider.name}`, href: provider.keyPage } };
  return (
    <form
      noValidate
      className="flex flex-col gap-4"
      onSubmit={(event) => {
        event.preventDefault();
        void form.handleSubmit();
      }}
    >
      <form.AppForm>
        <GuidedSteps
          label={`Steps to add your ${provider.name} key`}
          steps={[
            { label: open, ...link },
            { label: copy },
            { label: paste, children: <KeyField form={form} /> },
          ]}
        />
        <p className="text-meta text-ink-3 in-data-[density=touch]:text-body-touch">
          {passwordWords(provider)}
        </p>
        <DialogActions>
          <DialogClose>
            <Button>Cancel</Button>
          </DialogClose>
          <form.SubmitButton label="Check key" />
        </DialogActions>
      </form.AppForm>
    </form>
  );
}
