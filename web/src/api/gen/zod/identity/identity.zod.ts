import * as zod from "zod";

import {
  IdentityApiKeyMade,
  IdentityClaimBody,
  IdentityClaimed,
  IdentityPinChange,
  IdentityReconfirmation,
  IdentityRegeneratedAPIKey,
  IdentitySession,
  IdentitySettings,
  IdentitySettingsPatch,
  IdentitySignInBody,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary Change your PIN, or set a new one after a passkey sign-in on a turned-off PIN
 */
export const ChangePINBody = IdentityPinChange;

export const ChangePINResponse = zod.void();

/**
 * @summary Get when the API key was made
 */
export const GetAPIKeyResponse = IdentityApiKeyMade;

/**
 * @summary Replace the API key, shown this once; the old one stops working
 */
export const RegenerateAPIKeyResponse = IdentityRegeneratedAPIKey;

/**
 * @summary Get the sign-in limits and lifetimes, and the address links use
 */
export const GetIdentitySettingsResponse = IdentitySettings;

/**
 * @summary Change the sign-in limits and lifetimes, or the address links use
 */
export const UpdateIdentitySettingsBody = IdentitySettingsPatch;

export const UpdateIdentitySettingsResponse = IdentitySettings;

/**
 * @summary Sign out
 */
export const DeleteSessionResponse = zod.void();

/**
 * @summary Get the signed-in Account
 */
export const GetSessionResponse = IdentitySession;

/**
 * @summary Sign in with email and PIN
 */
export const CreateSessionBody = IdentitySignInBody;

export const CreateSessionResponse = IdentitySession;

/**
 * @summary Confirm it's you with your PIN, for actions that change who can get in
 */
export const ReconfirmSessionBody = IdentityReconfirmation;

export const ReconfirmSessionResponse = zod.void();

/**
 * @summary Create the first Admin and make this Node a Controller
 */
export const ClaimControllerBody = IdentityClaimBody;

export const ClaimControllerResponse = IdentityClaimed;
