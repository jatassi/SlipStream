import type { MailOutcome } from "../api/gen/model";
import { ReadOnlyField } from "../ui/read-only-field";

const mailSentence: {
  readonly [O in MailOutcome["outcome"]]: (to: string, detail: string) => string;
} = {
  sent: (to) => `Emailed it to ${to}.`,
  failed: (to, detail) =>
    `The email to ${to} failed, so copy the link and send it yourself. The mail server said: ${detail}`,
  "not-set-up": () => "Email is not set up, so copy the link and send it yourself.",
};

/** A link just issued: the URL to copy, and what became of its email. */
export function IssuedLink({
  label,
  url,
  to,
  mail,
}: {
  label: string;
  url: string;
  to: string;
  mail: MailOutcome;
}) {
  return (
    <div className="flex flex-col gap-4">
      <ReadOnlyField
        label={label}
        value={url}
        description="Anyone with this link can use it, so send it only to them."
      />
      <output className="text-ink-2">{mailSentence[mail.outcome](to, mail.detail ?? "")}</output>
    </div>
  );
}
