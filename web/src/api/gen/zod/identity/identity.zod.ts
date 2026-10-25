import * as zod from "zod";

import {
  AccountPatchInputBody,
  EmailChangeInputBody,
  IdentityAccountDetail,
  IdentityAccountRow,
  IdentityAccountRows,
  IdentityAddress,
  IdentityAddressChange,
  IdentityApiKeyMade,
  IdentityCeremony,
  IdentityClaimBody,
  IdentityClaimed,
  IdentityEmailChangeStarted,
  IdentityIssuedResetLink,
  IdentityIssuedSignupLink,
  IdentityLinkView,
  IdentityPasskey,
  IdentityPasskeyRows,
  IdentityPendingLinks,
  IdentityPinChange,
  IdentityReconfirmation,
  IdentityRegeneratedAPIKey,
  IdentitySession,
  IdentitySessionRows,
  IdentitySettings,
  IdentitySettingsPatch,
  IdentitySignInBody,
  IdentitySignedIn,
  IdentitySignupLinkURL,
  JoinInputBody,
  NewPINInputBody,
  NewSignupLinkInputBody,
  OwnPatchInputBody,
  PasskeyRenameInputBody,
  ResetLinkInputBody,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary Change your display name, or whether the passkey prompt may show
 */
export const UpdateOwnAccountBody = OwnPatchInputBody;

export const UpdateOwnAccountResponse = IdentitySession;

/**
 * @summary Change your email by sending a confirm link to the new address. The email changes when the link is followed
 */
export const StartEmailChangeBody = EmailChangeInputBody;

export const StartEmailChangeResponse = IdentityEmailChangeStarted;

/**
 * @summary Change your PIN, or set a new one after a passkey sign-in on a turned-off PIN
 */
export const ChangePINBody = IdentityPinChange;

export const ChangePINResponse = zod.void();

/**
 * @summary List every Account
 */
export const ListAccountsResponse = IdentityAccountRows;

/**
 * @summary Delete an Account with its sessions, passkeys and links
 */
export const DeleteAccountParams = zod.object({
  id: zod.uuid(),
});

export const DeleteAccountResponse = zod.void();

/**
 * @summary Get an Account with its passkeys and sessions
 */
export const GetAccountParams = zod.object({
  id: zod.uuid(),
});

export const GetAccountResponse = IdentityAccountDetail;

/**
 * @summary Change an Account's name, email or role, or turn it off or on
 */
export const UpdateAccountParams = zod.object({
  id: zod.uuid(),
});

export const UpdateAccountBody = AccountPatchInputBody;

export const UpdateAccountResponse = IdentityAccountRow;

/**
 * @summary Issue a Reset link for an Account and end its sessions. Only this response shows the link
 */
export const CreateResetLinkParams = zod.object({
  id: zod.uuid(),
});

export const CreateResetLinkBody = ResetLinkInputBody;

export const CreateResetLinkResponse = IdentityIssuedResetLink;

/**
 * @summary Sign an Account out everywhere
 */
export const EndAccountSessionsParams = zod.object({
  id: zod.uuid(),
});

export const EndAccountSessionsResponse = zod.void();

/**
 * @summary List the devices an Account is signed in on
 */
export const ListAccountSessionsParams = zod.object({
  id: zod.uuid(),
});

export const ListAccountSessionsResponse = IdentitySessionRows;

/**
 * @summary Sign one of an Account's devices out
 */
export const EndAccountSessionParams = zod.object({
  id: zod.uuid(),
  session: zod.uuid(),
});

export const EndAccountSessionResponse = zod.void();

/**
 * @summary Get when the API key was made
 */
export const GetAPIKeyResponse = IdentityApiKeyMade;

/**
 * @summary Replace the API key, shown this once. The old key stops working
 */
export const RegenerateAPIKeyResponse = IdentityRegeneratedAPIKey;

/**
 * @summary Get what a confirm link's page shows: the new email, or why it no longer works
 */
export const getEmailLinkByTokenPathTokenMax = 128;

export const GetEmailLinkByTokenParams = zod.object({
  token: zod.string().max(getEmailLinkByTokenPathTokenMax),
});

export const GetEmailLinkByTokenResponse = IdentityLinkView;

/**
 * @summary Change an Account's email to the one a confirm link proved
 */
export const confirmEmailPathTokenMax = 128;

export const ConfirmEmailParams = zod.object({
  token: zod.string().max(confirmEmailPathTokenMax),
});

export const ConfirmEmailResponse = zod.void();

/**
 * @summary Get the address links use, and the one Settings > Address suggests
 */
export const GetAddressResponse = IdentityAddress;

/**
 * @summary Set the address links use
 */
export const SetAddressBody = IdentityAddressChange;

export const SetAddressResponse = IdentityAddress;

/**
 * @summary Get the sign-in limits and lifetimes
 */
export const GetIdentitySettingsResponse = IdentitySettings;

/**
 * @summary Change the sign-in limits and lifetimes
 */
export const UpdateIdentitySettingsBody = IdentitySettingsPatch;

export const UpdateIdentitySettingsResponse = IdentitySettings;

/**
 * @summary List your passkeys at every address
 */
export const ListPasskeysResponse = IdentityPasskeyRows;

/**
 * @summary Start signing in with a passkey for this address
 */
export const StartPasskeySignInResponse = IdentityCeremony;

/**
 * @summary Sign in with the browser's passkey response. On a turned-off PIN, the session must set a new PIN first
 */
export const finishPasskeySignInPathCeremonyMax = 64;

export const FinishPasskeySignInParams = zod.object({
  ceremony: zod.string().max(finishPasskeySignInPathCeremonyMax),
});

export const FinishPasskeySignInBody = zod
  .unknown()
  .describe("The browser's PublicKeyCredential, as its toJSON() returns it");

export const FinishPasskeySignInResponse = IdentitySignedIn;

/**
 * @summary Start adding a passkey for this address
 */
export const StartPasskeyRegistrationResponse = IdentityCeremony;

/**
 * @summary Finish adding a passkey with the browser's response
 */
export const finishPasskeyRegistrationPathCeremonyMax = 64;

export const FinishPasskeyRegistrationParams = zod.object({
  ceremony: zod.string().max(finishPasskeyRegistrationPathCeremonyMax),
});

export const FinishPasskeyRegistrationBody = zod
  .unknown()
  .describe("The browser's PublicKeyCredential, as its toJSON() returns it");

export const FinishPasskeyRegistrationResponse = IdentityPasskey;

/**
 * @summary Delete one of your passkeys
 */
export const DeletePasskeyParams = zod.object({
  id: zod.uuid(),
});

export const DeletePasskeyResponse = zod.void();

/**
 * @summary Rename one of your passkeys
 */
export const UpdatePasskeyParams = zod.object({
  id: zod.uuid(),
});

export const UpdatePasskeyBody = PasskeyRenameInputBody;

export const UpdatePasskeyResponse = IdentityPasskey;

/**
 * @summary Get what a Reset link's page shows: who it is from, or why it no longer works
 */
export const getResetLinkByTokenPathTokenMax = 128;

export const GetResetLinkByTokenParams = zod.object({
  token: zod.string().max(getResetLinkByTokenPathTokenMax),
});

export const GetResetLinkByTokenResponse = IdentityLinkView;

/**
 * @summary Set a new PIN with a Reset link, ending every session, and sign in
 */
export const redeemResetLinkPathTokenMax = 128;

export const RedeemResetLinkParams = zod.object({
  token: zod.string().max(redeemResetLinkPathTokenMax),
});

export const RedeemResetLinkBody = NewPINInputBody;

export const RedeemResetLinkResponse = IdentitySignedIn;

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

export const CreateSessionResponse = IdentitySignedIn;

/**
 * @summary Confirm it's you with your PIN, for actions that change who can get in
 */
export const ReconfirmSessionBody = IdentityReconfirmation;

export const ReconfirmSessionResponse = zod.void();

/**
 * @summary Start confirming it's you with one of your passkeys for this address
 */
export const StartPasskeyReconfirmResponse = IdentityCeremony;

/**
 * @summary Confirm it's you with the browser's passkey response
 */
export const finishPasskeyReconfirmPathCeremonyMax = 64;

export const FinishPasskeyReconfirmParams = zod.object({
  ceremony: zod.string().max(finishPasskeyReconfirmPathCeremonyMax),
});

export const FinishPasskeyReconfirmBody = zod
  .unknown()
  .describe("The browser's PublicKeyCredential, as its toJSON() returns it");

export const FinishPasskeyReconfirmResponse = zod.void();

/**
 * @summary Sign out every device but this one
 */
export const EndOtherSessionsQueryParams = zod.object({
  others: zod.boolean().describe("Must be true: every session but this one ends"),
});

export const EndOtherSessionsResponse = zod.void();

/**
 * @summary List the devices your Account is signed in on
 */
export const ListSessionsResponse = IdentitySessionRows;

/**
 * @summary Sign one of your devices out
 */
export const EndSessionParams = zod.object({
  id: zod.uuid(),
});

export const EndSessionResponse = zod.void();

/**
 * @summary Create the first Admin and make this Node a Controller
 */
export const ClaimControllerBody = IdentityClaimBody;

export const ClaimControllerResponse = IdentityClaimed;

/**
 * @summary List the Signup links nobody has used yet
 */
export const ListSignupLinksResponse = IdentityPendingLinks;

/**
 * @summary Issue a Signup link for an email, email it, and return it to copy
 */
export const CreateSignupLinkBody = NewSignupLinkInputBody;

export const CreateSignupLinkResponse = IdentityIssuedSignupLink;

/**
 * @summary Get what a Signup link's page shows: who it is from, or why it no longer works
 */
export const getSignupLinkByTokenPathTokenMax = 128;

export const GetSignupLinkByTokenParams = zod.object({
  token: zod.string().max(getSignupLinkByTokenPathTokenMax),
});

export const GetSignupLinkByTokenResponse = IdentityLinkView;

/**
 * @summary Make the Member's Account with a Signup link and sign in
 */
export const redeemSignupLinkPathTokenMax = 128;

export const RedeemSignupLinkParams = zod.object({
  token: zod.string().max(redeemSignupLinkPathTokenMax),
});

export const RedeemSignupLinkBody = JoinInputBody;

export const RedeemSignupLinkResponse = IdentitySignedIn;

/**
 * @summary Cancel a pending Signup link
 */
export const DeleteSignupLinkParams = zod.object({
  id: zod.uuid(),
});

export const DeleteSignupLinkResponse = zod.void();

/**
 * @summary Get a pending Signup link with its URL, to copy again
 */
export const GetSignupLinkParams = zod.object({
  id: zod.uuid(),
});

export const GetSignupLinkResponse = IdentitySignupLinkURL;

/**
 * @summary Send a pending Signup link again as a new link. The old link stops working
 */
export const ResendSignupLinkParams = zod.object({
  id: zod.uuid(),
});

export const ResendSignupLinkResponse = IdentityIssuedSignupLink;
