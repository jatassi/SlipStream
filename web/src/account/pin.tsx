import { useState } from "react";

import { Button } from "../ui/button";
import { List, ListRow } from "../ui/list";
import { Section } from "../ui/section";
import { ChangePinDialog } from "./change-pin";

export function Pin() {
  const [open, setOpen] = useState(false);
  return (
    <Section title="PIN">
      <List label="PIN">
        <ListRow
          title="PIN"
          meta="Four digits that sign you in on any device"
          trailing={
            <Button
              size="small"
              onClick={() => {
                setOpen(true);
              }}
            >
              Change PIN
            </Button>
          }
        />
      </List>
      <ChangePinDialog open={open} onOpenChange={setOpen} />
    </Section>
  );
}
