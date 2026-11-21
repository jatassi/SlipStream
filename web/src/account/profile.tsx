import { useState } from "react";

import { useAccount } from "../session/session";
import { Fact, Facts } from "../ui/facts";
import { Section } from "../ui/section";
import { EmailDialog } from "./email-dialog";
import { KeyPromptSwitch } from "./key-prompt";
import { NameDialog } from "./name-dialog";

type Editing = "name" | "email" | null;

export function Profile() {
  const { displayName, email } = useAccount();
  const [editing, setEditing] = useState<Editing>(null);
  const openFor = (what: Exclude<Editing, null>) => (open: boolean) => {
    setEditing(open ? what : null);
  };
  const change = (what: Exclude<Editing, null>, name: string) => ({
    label: "Change",
    name,
    onClick: () => {
      setEditing(what);
    },
  });
  return (
    <Section title="Profile">
      <div className="flex flex-col gap-3">
        <Facts look="sheet">
          <Fact label="Display name" action={change("name", "Change display name")}>
            {displayName}
          </Fact>
          <Fact label="Email" action={change("email", "Change email")}>
            {email}
          </Fact>
        </Facts>
        <KeyPromptSwitch />
      </div>
      <NameDialog current={displayName} open={editing === "name"} onOpenChange={openFor("name")} />
      <EmailDialog open={editing === "email"} onOpenChange={openFor("email")} />
    </Section>
  );
}
