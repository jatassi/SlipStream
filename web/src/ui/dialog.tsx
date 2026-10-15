import { Dialog as BaseDialog } from "@base-ui/react/dialog";
import type { ReactElement, ReactNode } from "react";

import { usePortalDensity } from "./portal-density";
import { Sheet } from "./sheet";
import { useShell } from "./shell-context";

const widths = { 460: "w-[460px]", 680: "w-[680px]", 940: "w-[940px]" } as const;

export type DialogProps = {
  title: string;
  size?: keyof typeof widths;
  trigger?: ReactElement;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  children: ReactNode;
};

export function Dialog({ size, ...props }: DialogProps) {
  return useShell() === "phone" ? (
    <Sheet {...props} />
  ) : (
    <CentredDialog size={size ?? 460} {...props} />
  );
}

export function DialogClose({ children }: { children: ReactElement }) {
  return <BaseDialog.Close render={children} />;
}

const popup =
  "max-h-full max-w-full overflow-y-auto rounded-dialog bg-popover p-6 text-ink-1 shadow-float-large outline-none backdrop-blur-[30px] backdrop-saturate-[1.7] transition-[scale,opacity] duration-(--duration-dialog) ease-spring data-starting-style:opacity-0 data-ending-style:opacity-0 motion-safe:data-starting-style:scale-[.96] motion-safe:data-ending-style:scale-[.96]";

function CentredDialog({
  title,
  size,
  trigger,
  children,
  ...root
}: Omit<DialogProps, "size"> & { size: keyof typeof widths }) {
  const { density, probe } = usePortalDensity();
  return (
    <>
      {probe}
      <BaseDialog.Root {...root}>
        {trigger && <BaseDialog.Trigger render={trigger} />}
        <BaseDialog.Portal data-density={density}>
          <BaseDialog.Backdrop className="fixed inset-0 bg-[rgb(0_0_0/0.5)] transition-opacity duration-(--duration-dialog) ease-spring data-starting-style:opacity-0 data-ending-style:opacity-0" />
          <BaseDialog.Viewport className="fixed inset-0 grid place-items-center p-4">
            <BaseDialog.Popup className={`${popup} ${widths[size]}`}>
              <BaseDialog.Title className="mb-4 text-title">{title}</BaseDialog.Title>
              {children}
            </BaseDialog.Popup>
          </BaseDialog.Viewport>
        </BaseDialog.Portal>
      </BaseDialog.Root>
    </>
  );
}
