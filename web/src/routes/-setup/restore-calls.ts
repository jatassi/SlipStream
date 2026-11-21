import { isProblem } from "../../api/errors";
import { confirmRestore, openRestore, uploadRestore } from "../../api/gen/query/restore/restore";
import type { RestoreCalls } from "../../restore/flow";
import { forgetSetupCode, setupCodeHeaders } from "../../session/setup-code";
import type { FormProps } from "./form-props";

export function setupCalls({ code, restarted }: FormProps, claimed: () => void): RestoreCalls {
  const headers = setupCodeHeaders(code);
  return {
    upload: (file, onUploadProgress) => uploadRestore(file, { headers, onUploadProgress }),
    open: (id, passphrase) => openRestore(id, { passphrase }, { headers }),
    confirm: (id, body) => confirmRestore(id, body, { headers }),
    leaves: (error) => {
      if (isProblem(error, "setup_code_wrong")) {
        restarted();
        return true;
      }
      if (isProblem(error, "already_claimed")) {
        forgetSetupCode();
        claimed();
        return true;
      }
      return false;
    },
  };
}
