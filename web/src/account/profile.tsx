import { useState } from "react";

import { useAccount } from "../session/session";
import { Button } from "../ui/button";
import { List, ListRow } from "../ui/list";
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
  const change = (what: Exclude<Editing, null>, label: string) => (
    <Button
      size="small"
      aria-label={label}
      onClick={() => {
        setEditing(what);
      }}
    >
      Change
    </Button>
  );
  return (
    <Section title="Profile">
      <div className="flex flex-col gap-3">
        <List label="Profile">
          <ListRow
            title={displayName}
            meta="Display name"
            trailing={change("name", "Change display name")}
          />
          <ListRow title={email} meta="Email" trailing={change("email", "Change email")} />
        </List>
        <KeyPromptSwitch />
      </div>
      <NameDialog current={displayName} open={editing === "name"} onOpenChange={openFor("name")} />
      <EmailDialog open={editing === "email"} onOpenChange={openFor("email")} />
    </Section>
  );
}
