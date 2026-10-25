import type { MailOutcome } from "../api/gen/model";
import { ReadOnlyField } from "../ui/read-only-field";

const mailSentence: {
  readonly [O in MailOutcome["outcome"]]: (to: string, detail: string) => string;
} = {
  sent: (to) => `Emailed it to ${to}.`,
  failed: (to, detail) =>
    `The email to ${to} failed. Copy the link and send it yourself. The mail server responded: ${detail}`,
  "not-set-up": () => "Email isn't set up. Copy the link and send it yourself.",
};

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
        description="Anyone with this link can use it. Send it only to the person it's for."
      />
      <output className="text-ink-2">{mailSentence[mail.outcome](to, mail.detail ?? "")}</output>
    </div>
  );
}
