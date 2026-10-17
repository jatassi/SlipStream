import { useRouter } from "@tanstack/react-router";
import { z } from "zod";

import type { MailOutcome, MailSettings, MailSettingsPatch } from "../api/gen/model";
import {
  updateMailSettings,
  useGetMailSettingsSuspense,
  useSendTestMail,
} from "../api/gen/query/mail/mail";
import { useAccount } from "../session/session";
import { Button } from "../ui/button";
import { submitErrors, useAppForm } from "../ui/form";
import { Select, type SelectOption } from "../ui/select";
import { toast } from "../ui/toast";

type Security = MailSettings["security"];

const securityOptions: readonly SelectOption<Security>[] = [
  { value: "none", label: "None" },
  { value: "starttls", label: "STARTTLS" },
  { value: "tls", label: "TLS" },
];

const mailForm = z.object({
  host: z.string().trim(),
  port: z
    .string()
    .trim()
    .regex(/^[1-9][0-9]{0,4}$/u, "Enter a port number"),
  security: z.enum(["none", "starttls", "tls"]),
  username: z.string().trim(),
  password: z.string(),
  fromAddress: z.union([z.literal(""), z.email("Enter an email address")]),
  fromName: z.string().trim(),
});

type MailForm = z.input<typeof mailForm>;

const formOf = (settings: MailSettings): MailForm => ({
  host: settings.host,
  port: String(settings.port),
  security: settings.security,
  username: settings.username,
  password: "",
  fromAddress: settings.fromAddress,
  fromName: settings.fromName,
});

function patchOf(value: MailForm): MailSettingsPatch {
  const { port, password, ...rest } = mailForm.parse(value);
  return { ...rest, port: Number(port), ...(password === "" ? {} : { password }) };
}

const testSentence: {
  readonly [O in MailOutcome["outcome"]]: (to: string, detail: string) => string;
} = {
  sent: (to) => `Sent a test email to ${to}.`,
  failed: (_to, detail) => `The test email failed. The mail server said: ${detail}`,
  "not-set-up": () => "Email is not set up. Save a server first.",
};

function TestEmail() {
  const { email } = useAccount();
  const test = useSendTestMail();
  return (
    <div className="flex flex-col items-start gap-3">
      <Button
        busy={test.isPending}
        onClick={() => {
          test.mutate();
        }}
      >
        Send a test email
      </Button>
      {test.data === undefined ? null : (
        <output className="text-ink-2">
          {testSentence[test.data.outcome](email, test.data.detail ?? "")}
        </output>
      )}
      {test.error === null ? null : (
        <p role="alert" className="text-fail-ink">
          {test.error.message}
        </p>
      )}
    </div>
  );
}

function useMailForm(settings: MailSettings) {
  const router = useRouter();
  return useAppForm({
    defaultValues: formOf(settings),
    validators: { onSubmit: mailForm },
    onSubmit: async ({ value, formApi }) => {
      const saved = await updateMailSettings(patchOf(value)).catch((error: unknown) => {
        formApi.setErrorMap({ onSubmit: submitErrors(error) });
        return null;
      });
      if (saved !== null) {
        formApi.reset(formOf(saved));
        toast("Saved the email settings");
        await router.invalidate();
      }
    },
  });
}

type TextName = Exclude<keyof MailForm, "security">;

type TextSpec = {
  readonly name: TextName;
  readonly label: string;
  readonly type?: "email" | "password";
  readonly placeholder?: string;
  readonly inputMode?: "numeric";
  readonly description?: string;
};

type FieldSpec = TextSpec | { readonly name: "security" };

const fieldsOf = (passwordSet: boolean): readonly FieldSpec[] => [
  { name: "host", label: "Server", placeholder: "smtp.example.com" },
  { name: "port", label: "Port", inputMode: "numeric" },
  { name: "security" },
  {
    name: "username",
    label: "Username",
    description: "Leave empty when the server takes no sign-in.",
  },
  {
    name: "password",
    label: "Password",
    type: "password",
    description: passwordSet
      ? "A password is saved. Type a new one to replace it."
      : "No password is saved.",
  },
  {
    name: "fromAddress",
    label: "From address",
    type: "email",
    placeholder: "slipstream@example.com",
  },
  { name: "fromName", label: "From name" },
];

function MailFormFields({ settings }: { settings: MailSettings }) {
  const form = useMailForm(settings);
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
        {fieldsOf(settings.passwordSet).map((spec) =>
          spec.name === "security" ? (
            <form.Field key={spec.name} name={spec.name}>
              {(field) => (
                <Select
                  label="Security"
                  options={securityOptions}
                  value={field.state.value}
                  onValueChange={field.handleChange}
                />
              )}
            </form.Field>
          ) : (
            <form.AppField key={spec.name} name={spec.name}>
              {(field) => (
                <field.TextField
                  {...spec}
                  autoComplete={spec.type === "password" ? "new-password" : "off"}
                  spellCheck={false}
                />
              )}
            </form.AppField>
          ),
        )}
        <form.FormAlert />
        <div>
          <form.SubmitButton label="Save" />
        </div>
      </form.AppForm>
    </form>
  );
}

export function EmailSection() {
  const settings = useGetMailSettingsSuspense().data;
  return (
    <div className="flex flex-col gap-6">
      <p className="text-ink-2">
        {settings.host === ""
          ? "Not set up. SlipStream sends no email until a server is saved."
          : `Every email SlipStream sends goes through ${settings.host}.`}
      </p>
      <MailFormFields settings={settings} />
      <TestEmail />
    </div>
  );
}
