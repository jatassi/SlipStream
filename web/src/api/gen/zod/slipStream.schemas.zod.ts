import * as zod from "zod";

export const accountPatchInputBodyDisplayNameMax = 100;

export const accountPatchInputBodyEmailMax = 254;

export const AccountPatchInputBody = zod.object({
  admin: zod.boolean().exactOptional(),
  disabled: zod.boolean().exactOptional().describe("Turning it off ends every session it holds"),
  displayName: zod.string().min(1).max(accountPatchInputBodyDisplayNameMax).exactOptional(),
  email: zod
    .email()
    .max(accountPatchInputBodyEmailMax)
    .exactOptional()
    .describe("Changed at once; the old address is told"),
});

export type AccountPatchInputBody = zod.input<typeof AccountPatchInputBody>;
export type AccountPatchInputBodyOutput = zod.output<typeof AccountPatchInputBody>;

export const emailChangeInputBodyEmailMax = 254;

export const EmailChangeInputBody = zod.object({
  email: zod.email().max(emailChangeInputBodyEmailMax),
});

export type EmailChangeInputBody = zod.input<typeof EmailChangeInputBody>;
export type EmailChangeInputBodyOutput = zod.output<typeof EmailChangeInputBody>;

export const FieldError = zod.object({
  location: zod.string(),
  message: zod.string(),
});

export type FieldError = zod.input<typeof FieldError>;
export type FieldErrorOutput = zod.output<typeof FieldError>;

export const FilesMissingCell = zod.object({
  edition: zod.enum(["standard"]),
  itemId: zod.uuid(),
  state: zod.enum(["missing"]),
  targetId: zod.uuid(),
  targetName: zod.string(),
  wanted: zod.boolean().describe("Missing or failed, and its Item monitored"),
});

export type FilesMissingCell = zod.input<typeof FilesMissingCell>;
export type FilesMissingCellOutput = zod.output<typeof FilesMissingCell>;

export const FilesDownloadingCell = zod.object({
  edition: zod.enum(["standard"]),
  itemId: zod.uuid(),
  state: zod.enum(["downloading"]),
  targetId: zod.uuid(),
  targetName: zod.string(),
  wanted: zod.boolean().describe("Missing or failed, and its Item monitored"),
});

export type FilesDownloadingCell = zod.input<typeof FilesDownloadingCell>;
export type FilesDownloadingCellOutput = zod.output<typeof FilesDownloadingCell>;

export const FilesFailedCell = zod.object({
  edition: zod.enum(["standard"]),
  itemId: zod.uuid(),
  state: zod.enum(["failed"]),
  targetId: zod.uuid(),
  targetName: zod.string(),
  wanted: zod.boolean().describe("Missing or failed, and its Item monitored"),
});

export type FilesFailedCell = zod.input<typeof FilesFailedCell>;
export type FilesFailedCellOutput = zod.output<typeof FilesFailedCell>;

export const filesProbeHeightMin = 0;

export const filesProbeWidthMin = 0;

export const FilesProbe = zod.object({
  height: zod.int().min(filesProbeHeightMin),
  width: zod.int().min(filesProbeWidthMin),
});

export type FilesProbe = zod.input<typeof FilesProbe>;
export type FilesProbeOutput = zod.output<typeof FilesProbe>;

export const filesLibraryFileSizeMin = 0;

export const FilesLibraryFile = zod.object({
  id: zod.uuid(),
  path: zod.string().describe("Relative to its root, / separated"),
  probe: FilesProbe,
  size: zod.int().min(filesLibraryFileSizeMin),
});

export type FilesLibraryFile = zod.input<typeof FilesLibraryFile>;
export type FilesLibraryFileOutput = zod.output<typeof FilesLibraryFile>;

export const FilesMetCell = zod.object({
  edition: zod.enum(["standard"]),
  file: FilesLibraryFile.describe("The file filling it"),
  itemId: zod.uuid(),
  state: zod.enum(["met"]),
  targetId: zod.uuid(),
  targetName: zod.string(),
  wanted: zod.boolean().describe("Missing or failed, and its Item monitored"),
});

export type FilesMetCell = zod.input<typeof FilesMetCell>;
export type FilesMetCellOutput = zod.output<typeof FilesMetCell>;

export const FilesCell = zod.discriminatedUnion("state", [
  FilesMissingCell,
  FilesDownloadingCell,
  FilesFailedCell,
  FilesMetCell,
]);

export type FilesCell = zod.input<typeof FilesCell>;
export type FilesCellOutput = zod.output<typeof FilesCell>;

export const FilesCellList = zod.object({
  cells: zod.array(FilesCell).describe("By Item in tree order, then Target in Profile order"),
});

export type FilesCellList = zod.input<typeof FilesCellList>;
export type FilesCellListOutput = zod.output<typeof FilesCellList>;

export const FilesChanged = zod.object({
  workId: zod.uuid(),
});

export type FilesChanged = zod.input<typeof FilesChanged>;
export type FilesChangedOutput = zod.output<typeof FilesChanged>;

export const GrabsChanged = zod.object({
  grabId: zod.uuid(),
  workId: zod.uuid(),
});

export type GrabsChanged = zod.input<typeof GrabsChanged>;
export type GrabsChangedOutput = zod.output<typeof GrabsChanged>;

export const GrabsEntered = zod.object({
  at: zod.iso.datetime({ offset: true }),
  stage: zod.enum([
    "fetching",
    "downloading",
    "probing",
    "transferring",
    "importing",
    "imported",
    "failed",
  ]),
});

export type GrabsEntered = zod.input<typeof GrabsEntered>;
export type GrabsEnteredOutput = zod.output<typeof GrabsEntered>;

export const GrabsFailure = zod.object({
  code: zod.enum([
    "indexer_unavailable",
    "torrent_unreachable",
    "torrent_invalid",
    "infohash_mismatch",
    "torrent_v2_only",
    "no_download_client",
    "download_failed",
    "no_video_file",
    "probe_failed",
    "fills_nothing",
    "transfer_failed",
    "import_failed",
  ]),
  message: zod.string(),
  stage: zod.enum(["fetching", "downloading", "probing", "transferring", "importing"]),
});

export type GrabsFailure = zod.input<typeof GrabsFailure>;
export type GrabsFailureOutput = zod.output<typeof GrabsFailure>;

export const grabsProgressDoneMin = 0;

export const grabsProgressTotalMin = 0;

export const GrabsProgress = zod.object({
  done: zod
    .int()
    .min(grabsProgressDoneMin)
    .describe("Bytes downloaded, or received and verified while transferring"),
  total: zod.int().min(grabsProgressTotalMin).describe("Bytes wanted"),
});

export type GrabsProgress = zod.input<typeof GrabsProgress>;
export type GrabsProgressOutput = zod.output<typeof GrabsProgress>;

export const grabsGrabReleaseSizeMin = 0;

export const GrabsGrab = zod.object({
  createdAt: zod.iso.datetime({ offset: true }),
  edition: zod.enum(["standard"]),
  failure: GrabsFailure.exactOptional().describe("Why it failed, absent unless it did"),
  history: zod.array(GrabsEntered).describe("Each stage it entered, oldest first"),
  id: zod.uuid(),
  indexerId: zod.uuid(),
  infoHash: zod
    .string()
    .nullable()
    .describe("The v1 info hash in hex, null until the .torrent is fetched"),
  itemId: zod.uuid(),
  nodeId: zod.uuid().describe("The Node that downloads it: the Indexer's home Node"),
  progress: GrabsProgress.exactOptional().describe(
    "The download's or the Transfer's progress, absent unless it is downloading and its Node reported it, or transferring",
  ),
  releaseSize: zod.int().min(grabsGrabReleaseSizeMin),
  releaseTitle: zod.string(),
  stage: zod.enum([
    "fetching",
    "downloading",
    "probing",
    "transferring",
    "importing",
    "imported",
    "failed",
  ]),
  targetIds: zod.array(zod.string()).describe("The Targets it was grabbed for, in Profile order"),
  workId: zod.uuid(),
});

export type GrabsGrab = zod.input<typeof GrabsGrab>;
export type GrabsGrabOutput = zod.output<typeof GrabsGrab>;

export const GrabsGrabList = zod.object({
  grabs: zod.array(GrabsGrab).describe("Newest first"),
});

export type GrabsGrabList = zod.input<typeof GrabsGrabList>;
export type GrabsGrabListOutput = zod.output<typeof GrabsGrabList>;

export const grabsSearchCandidatesMin = 0;

export const GrabsSearch = zod.object({
  candidates: zod.int().min(grabsSearchCandidatesMin).describe("The releases the Indexer answered"),
  finishedAt: zod.iso.datetime({ offset: true }).nullable().describe("Null while the run searches"),
  grabId: zod.uuid().nullable().describe("The Grab the run made, null when none"),
  id: zod.uuid(),
  indexerId: zod.uuid(),
  itemId: zod.uuid(),
  outcome: zod
    .union([
      zod.literal("grabbed"),
      zod.literal("nothing_acceptable"),
      zod.literal("indexer_unavailable"),
      zod.literal("indexer_refused"),
      zod.literal(null),
    ])
    .nullable()
    .describe("Null while the run searches"),
  startedAt: zod.iso.datetime({ offset: true }),
});

export type GrabsSearch = zod.input<typeof GrabsSearch>;
export type GrabsSearchOutput = zod.output<typeof GrabsSearch>;

export const GrabsSearchChanged = zod.object({
  searchId: zod.uuid(),
  workId: zod.uuid(),
});

export type GrabsSearchChanged = zod.input<typeof GrabsSearchChanged>;
export type GrabsSearchChangedOutput = zod.output<typeof GrabsSearchChanged>;

export const GrabsSearchList = zod.object({
  searches: zod.array(GrabsSearch).describe("Newest first"),
});

export type GrabsSearchList = zod.input<typeof GrabsSearchList>;
export type GrabsSearchListOutput = zod.output<typeof GrabsSearchList>;

export const GrabsSearchStarted = zod.object({
  searchIds: zod.array(zod.string()).describe("One run per Item with a wanted cell"),
});

export type GrabsSearchStarted = zod.input<typeof GrabsSearchStarted>;
export type GrabsSearchStartedOutput = zod.output<typeof GrabsSearchStarted>;

export const identityAccountDisplayNameMax = 100;

export const identityAccountEmailMax = 254;

export const IdentityAccount = zod.object({
  admin: zod.boolean(),
  displayName: zod.string().min(1).max(identityAccountDisplayNameMax),
  email: zod.email().max(identityAccountEmailMax),
  id: zod.uuid(),
});

export type IdentityAccount = zod.input<typeof IdentityAccount>;
export type IdentityAccountOutput = zod.output<typeof IdentityAccount>;

export const IdentityPinState = zod.object({
  state: zod.enum(["fine", "waiting", "off"]),
  until: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("When the wait ends, while it waits"),
});

export type IdentityPinState = zod.input<typeof IdentityPinState>;
export type IdentityPinStateOutput = zod.output<typeof IdentityPinState>;

export const identityAccountRowDisplayNameMax = 100;

export const identityAccountRowEmailMax = 254;

export const IdentityAccountRow = zod.object({
  admin: zod.boolean(),
  disabled: zod.boolean().describe("Turned off: it can't sign in and holds no session"),
  displayName: zod.string().min(1).max(identityAccountRowDisplayNameMax),
  email: zod.email().max(identityAccountRowEmailMax),
  id: zod.uuid(),
  lastSignInAt: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("Absent until it first signs in"),
  passkeyCount: zod.int(),
  pin: IdentityPinState,
});

export type IdentityAccountRow = zod.input<typeof IdentityAccountRow>;
export type IdentityAccountRowOutput = zod.output<typeof IdentityAccountRow>;

export const IdentityPasskey = zod.object({
  createdAt: zod.iso.datetime({ offset: true }),
  host: zod.string().describe("The address's host the passkey signs in at"),
  id: zod.uuid(),
  lastUsedAt: zod.iso.datetime({ offset: true }).exactOptional(),
  name: zod.string(),
});

export type IdentityPasskey = zod.input<typeof IdentityPasskey>;
export type IdentityPasskeyOutput = zod.output<typeof IdentityPasskey>;

export const IdentitySessionRow = zod.object({
  address: zod.string().describe("The client address it signed in from"),
  createdAt: zod.iso.datetime({ offset: true }),
  current: zod.boolean().describe("The session making this request"),
  device: zod.string().describe("Such as Safari on iPhone, from the browser at sign-in"),
  id: zod.uuid(),
  lastSeenAt: zod.iso.datetime({ offset: true }),
});

export type IdentitySessionRow = zod.input<typeof IdentitySessionRow>;
export type IdentitySessionRowOutput = zod.output<typeof IdentitySessionRow>;

export const IdentityAccountDetail = zod.object({
  account: IdentityAccountRow,
  passkeys: zod.array(IdentityPasskey),
  sessions: zod.array(IdentitySessionRow),
});

export type IdentityAccountDetail = zod.input<typeof IdentityAccountDetail>;
export type IdentityAccountDetailOutput = zod.output<typeof IdentityAccountDetail>;

export const IdentityAccountRows = zod.object({
  accounts: zod.array(IdentityAccountRow),
});

export type IdentityAccountRows = zod.input<typeof IdentityAccountRows>;
export type IdentityAccountRowsOutput = zod.output<typeof IdentityAccountRows>;

export const IdentityAccountsChanged = zod.object({});

export type IdentityAccountsChanged = zod.input<typeof IdentityAccountsChanged>;
export type IdentityAccountsChangedOutput = zod.output<typeof IdentityAccountsChanged>;

export const IdentityAddress = zod.object({
  address: zod.string().describe("The origin links are built on, empty until set"),
  suggestedAddress: zod
    .string()
    .describe(
      "The address this request reached SlipStream at, or this machine's name when that was an IP address",
    ),
});

export type IdentityAddress = zod.input<typeof IdentityAddress>;
export type IdentityAddressOutput = zod.output<typeof IdentityAddress>;

export const identityAddressChangeAddressMax = 2048;

export const IdentityAddressChange = zod.object({
  address: zod.string().max(identityAddressChangeAddressMax),
});

export type IdentityAddressChange = zod.input<typeof IdentityAddressChange>;
export type IdentityAddressChangeOutput = zod.output<typeof IdentityAddressChange>;

export const IdentityApiKeyMade = zod.object({
  createdAt: zod.iso.datetime({ offset: true }),
});

export type IdentityApiKeyMade = zod.input<typeof IdentityApiKeyMade>;
export type IdentityApiKeyMadeOutput = zod.output<typeof IdentityApiKeyMade>;

export const IdentityCeremony = zod.object({
  ceremony: zod.string().describe("The id the finish carries in its path"),
  options: zod
    .unknown()
    .describe(
      "PublicKeyCredentialCreationOptionsJSON for a registration, PublicKeyCredentialRequestOptionsJSON for an assertion: the browser's parseCreationOptionsFromJSON or parseRequestOptionsFromJSON takes it",
    ),
});

export type IdentityCeremony = zod.input<typeof IdentityCeremony>;
export type IdentityCeremonyOutput = zod.output<typeof IdentityCeremony>;

export const identityClaimBodyDisplayNameMax = 100;

export const identityClaimBodyEmailMax = 254;

export const identityClaimBodyPinRegExp = new RegExp("^[0-9]{4}$");

export const IdentityClaimBody = zod.object({
  displayName: zod.string().min(1).max(identityClaimBodyDisplayNameMax),
  email: zod.email().max(identityClaimBodyEmailMax),
  pin: zod.string().regex(identityClaimBodyPinRegExp),
});

export type IdentityClaimBody = zod.input<typeof IdentityClaimBody>;
export type IdentityClaimBodyOutput = zod.output<typeof IdentityClaimBody>;

export const identityClaimedApiKeyRegExp = new RegExp("^ssk_[A-Za-z0-9_-]{43}$");

export const IdentityClaimed = zod.object({
  account: IdentityAccount,
  apiKey: zod.string().regex(identityClaimedApiKeyRegExp),
});

export type IdentityClaimed = zod.input<typeof IdentityClaimed>;
export type IdentityClaimedOutput = zod.output<typeof IdentityClaimed>;

export const IdentityDeadLink = zod.object({
  adminName: zod
    .string()
    .exactOptional()
    .describe(
      "Who to ask: absent for unknown, which names nobody, and for a link the API key or the host command issued",
    ),
  expiresAt: zod.iso
    .datetime({ offset: true })
    .exactOptional()
    .describe("Every reason but unknown"),
  issuedAt: zod.iso.datetime({ offset: true }).exactOptional().describe("Every reason but unknown"),
  reason: zod.enum(["unknown", "withdrawn", "expired", "used"]),
  state: zod.enum(["dead"]),
});

export type IdentityDeadLink = zod.input<typeof IdentityDeadLink>;
export type IdentityDeadLinkOutput = zod.output<typeof IdentityDeadLink>;

export const MailOutcome = zod.object({
  detail: zod
    .string()
    .exactOptional()
    .describe("The mail server's reply, or why none answered, when the email failed"),
  outcome: zod.enum(["sent", "failed", "not-set-up"]),
});

export type MailOutcome = zod.input<typeof MailOutcome>;
export type MailOutcomeOutput = zod.output<typeof MailOutcome>;

export const IdentityEmailChangeStarted = zod.object({
  email: MailOutcome.describe("What became of the confirm link's email to the new address"),
});

export type IdentityEmailChangeStarted = zod.input<typeof IdentityEmailChangeStarted>;
export type IdentityEmailChangeStartedOutput = zod.output<typeof IdentityEmailChangeStarted>;

export const IdentityIssuedResetLink = zod.object({
  email: MailOutcome,
  url: zod.url().describe("The link itself, shown only in this answer"),
});

export type IdentityIssuedResetLink = zod.input<typeof IdentityIssuedResetLink>;
export type IdentityIssuedResetLinkOutput = zod.output<typeof IdentityIssuedResetLink>;

export const IdentityPendingLink = zod.object({
  email: zod.email(),
  expiresAt: zod.iso.datetime({ offset: true }),
  id: zod.uuid(),
  issuedAt: zod.iso.datetime({ offset: true }),
  issuedBy: zod
    .string()
    .exactOptional()
    .describe("The Admin who issued it; absent for the API key"),
});

export type IdentityPendingLink = zod.input<typeof IdentityPendingLink>;
export type IdentityPendingLinkOutput = zod.output<typeof IdentityPendingLink>;

export const IdentityIssuedSignupLink = zod.object({
  email: MailOutcome,
  link: IdentityPendingLink,
  url: zod.url().describe("The link itself, to copy"),
});

export type IdentityIssuedSignupLink = zod.input<typeof IdentityIssuedSignupLink>;
export type IdentityIssuedSignupLinkOutput = zod.output<typeof IdentityIssuedSignupLink>;

export const IdentityLiveLink = zod.object({
  adminName: zod
    .string()
    .exactOptional()
    .describe("Who issued the link; absent when the API key or the host command did"),
  email: zod
    .email()
    .describe("Where the link went: the Account's email, or the new one it confirms"),
  expiresAt: zod.iso.datetime({ offset: true }),
  serverName: zod.string(),
  state: zod.enum(["live"]),
});

export type IdentityLiveLink = zod.input<typeof IdentityLiveLink>;
export type IdentityLiveLinkOutput = zod.output<typeof IdentityLiveLink>;

export const IdentityLinkView = zod.discriminatedUnion("state", [
  IdentityLiveLink,
  IdentityDeadLink,
]);

export type IdentityLinkView = zod.input<typeof IdentityLinkView>;
export type IdentityLinkViewOutput = zod.output<typeof IdentityLinkView>;

export const IdentityLinksChanged = zod.object({});

export type IdentityLinksChanged = zod.input<typeof IdentityLinksChanged>;
export type IdentityLinksChangedOutput = zod.output<typeof IdentityLinksChanged>;

export const IdentityPasskeyRows = zod.object({
  passkeys: zod.array(IdentityPasskey),
});

export type IdentityPasskeyRows = zod.input<typeof IdentityPasskeyRows>;
export type IdentityPasskeyRowsOutput = zod.output<typeof IdentityPasskeyRows>;

export const IdentityPasskeysChanged = zod.object({
  account: zod.uuid(),
});

export type IdentityPasskeysChanged = zod.input<typeof IdentityPasskeysChanged>;
export type IdentityPasskeysChangedOutput = zod.output<typeof IdentityPasskeysChanged>;

export const IdentityPendingLinks = zod.object({
  links: zod.array(IdentityPendingLink),
});

export type IdentityPendingLinks = zod.input<typeof IdentityPendingLinks>;
export type IdentityPendingLinksOutput = zod.output<typeof IdentityPendingLinks>;

export const identityPinChangeCurrentRegExp = new RegExp("^[0-9]{4}$");
export const identityPinChangeNewRegExp = new RegExp("^[0-9]{4}$");

export const IdentityPinChange = zod.object({
  current: zod
    .string()
    .regex(identityPinChangeCurrentRegExp)
    .exactOptional()
    .describe("The current PIN, left out only by a session that must set a new PIN"),
  new: zod.string().regex(identityPinChangeNewRegExp),
});

export type IdentityPinChange = zod.input<typeof IdentityPinChange>;
export type IdentityPinChangeOutput = zod.output<typeof IdentityPinChange>;

export const identityReconfirmationPinRegExp = new RegExp("^[0-9]{4}$");

export const IdentityReconfirmation = zod.object({
  pin: zod.string().regex(identityReconfirmationPinRegExp),
});

export type IdentityReconfirmation = zod.input<typeof IdentityReconfirmation>;
export type IdentityReconfirmationOutput = zod.output<typeof IdentityReconfirmation>;

export const identityRegeneratedAPIKeyApiKeyRegExp = new RegExp("^ssk_[A-Za-z0-9_-]{43}$");

export const IdentityRegeneratedAPIKey = zod.object({
  apiKey: zod.string().regex(identityRegeneratedAPIKeyApiKeyRegExp),
  createdAt: zod.iso.datetime({ offset: true }),
});

export type IdentityRegeneratedAPIKey = zod.input<typeof IdentityRegeneratedAPIKey>;
export type IdentityRegeneratedAPIKeyOutput = zod.output<typeof IdentityRegeneratedAPIKey>;

export const IdentitySession = zod.object({
  account: IdentityAccount,
  keyPrompt: zod
    .enum(["ask", "never"])
    .describe("Whether the prompt to add a passkey may show after sign-in"),
  pinRequired: zod
    .boolean()
    .describe(
      "The session must set a new PIN before anything but reading the session and signing out",
    ),
});

export type IdentitySession = zod.input<typeof IdentitySession>;
export type IdentitySessionOutput = zod.output<typeof IdentitySession>;

export const IdentitySessionRows = zod.object({
  sessions: zod.array(IdentitySessionRow),
});

export type IdentitySessionRows = zod.input<typeof IdentitySessionRows>;
export type IdentitySessionRowsOutput = zod.output<typeof IdentitySessionRows>;

export const IdentitySessionsChanged = zod.object({
  account: zod.uuid(),
});

export type IdentitySessionsChanged = zod.input<typeof IdentitySessionsChanged>;
export type IdentitySessionsChangedOutput = zod.output<typeof IdentitySessionsChanged>;

export const SettingNoun = zod.object({
  many: zod.string(),
  one: zod.string(),
});

export type SettingNoun = zod.input<typeof SettingNoun>;
export type SettingNounOutput = zod.output<typeof SettingNoun>;

export const SettingCountField = zod.object({
  applies: zod.enum(["live", "restart"]),
  default: zod.int(),
  defaultFrom: zod
    .string()
    .exactOptional()
    .describe("Where the default comes from, when it is another setting's value"),
  floor: zod
    .string()
    .exactOptional()
    .describe("Why the minimum protects someone outside the install"),
  isDefault: zod
    .boolean()
    .describe("The setting has no override; Reset to default leaves it unchanged"),
  key: zod.string(),
  kind: zod.enum(["count"]),
  label: zod.string(),
  lockedBy: zod
    .string()
    .exactOptional()
    .describe("The environment variable that sets the value; the control is read-only"),
  max: zod.int(),
  min: zod.int(),
  pendingRestart: zod
    .boolean()
    .describe("The saved value differs from the running one until the Node restarts"),
  running: zod.int(),
  source: zod.enum(["database", "default", "config.yaml", ".env", "env"]),
  unit: SettingNoun,
  value: zod.int(),
  zero: zod.string().exactOptional().describe("The words for 0, such as No limit"),
});

export type SettingCountField = zod.input<typeof SettingCountField>;
export type SettingCountFieldOutput = zod.output<typeof SettingCountField>;

export const SettingDurationField = zod.object({
  applies: zod.enum(["live", "restart"]),
  default: zod.int().describe("Seconds"),
  defaultFrom: zod
    .string()
    .exactOptional()
    .describe("Where the default comes from, when it is another setting's value"),
  floor: zod
    .string()
    .exactOptional()
    .describe("Why the minimum protects someone outside the install"),
  isDefault: zod
    .boolean()
    .describe("The setting has no override; Reset to default leaves it unchanged"),
  key: zod.string(),
  kind: zod.enum(["duration"]),
  label: zod.string(),
  lockedBy: zod
    .string()
    .exactOptional()
    .describe("The environment variable that sets the value; the control is read-only"),
  max: zod.int().describe("Seconds"),
  min: zod.int().describe("Seconds"),
  pendingRestart: zod
    .boolean()
    .describe("The saved value differs from the running one until the Node restarts"),
  running: zod.int().describe("Seconds"),
  source: zod.enum(["database", "default", "config.yaml", ".env", "env"]),
  step: zod.int().describe("Seconds: a second, minute, hour or day"),
  value: zod.int().describe("Seconds"),
});

export type SettingDurationField = zod.input<typeof SettingDurationField>;
export type SettingDurationFieldOutput = zod.output<typeof SettingDurationField>;

export const IdentitySettings = zod.object({
  addressFailures: SettingCountField,
  addressWindow: SettingDurationField,
  emailLinkLifetime: SettingDurationField,
  pinOffAfter: SettingCountField,
  pinWait: SettingDurationField,
  pinWaitAfter: SettingCountField,
  reconfirmWindow: SettingDurationField,
  resetLinkLifetime: SettingDurationField,
  sessionLifetime: SettingDurationField,
  signupLinkLifetime: SettingDurationField,
});

export type IdentitySettings = zod.input<typeof IdentitySettings>;
export type IdentitySettingsOutput = zod.output<typeof IdentitySettings>;

export const IdentitySettingsPatch = zod.object({
  addressFailures: zod.int().nullish().describe("null resets it to its default"),
  addressWindow: zod.int().nullish().describe("Seconds; null resets it to its default"),
  emailLinkLifetime: zod.int().nullish().describe("Seconds; null resets it to its default"),
  pinOffAfter: zod.int().nullish().describe("null resets it to its default"),
  pinWait: zod.int().nullish().describe("Seconds; null resets it to its default"),
  pinWaitAfter: zod.int().nullish().describe("null resets it to its default"),
  reconfirmWindow: zod.int().nullish().describe("Seconds; null resets it to its default"),
  resetLinkLifetime: zod.int().nullish().describe("Seconds; null resets it to its default"),
  sessionLifetime: zod.int().nullish().describe("Seconds; null resets it to its default"),
  signupLinkLifetime: zod.int().nullish().describe("Seconds; null resets it to its default"),
});

export type IdentitySettingsPatch = zod.input<typeof IdentitySettingsPatch>;
export type IdentitySettingsPatchOutput = zod.output<typeof IdentitySettingsPatch>;

export const identitySignInBodyEmailMax = 254;

export const identitySignInBodyPinRegExp = new RegExp("^[0-9]{4}$");

export const IdentitySignInBody = zod.object({
  email: zod.email().max(identitySignInBodyEmailMax),
  pin: zod.string().regex(identitySignInBodyPinRegExp),
});

export type IdentitySignInBody = zod.input<typeof IdentitySignInBody>;
export type IdentitySignInBodyOutput = zod.output<typeof IdentitySignInBody>;

export const IdentitySignedIn = zod.object({
  account: IdentityAccount,
  keyPrompt: zod
    .enum(["ask", "never"])
    .describe("Whether the prompt to add a passkey may show after sign-in"),
  passkeyPrompt: zod
    .boolean()
    .describe("Show the prompt to add a passkey for this address, once in this session"),
  pinRequired: zod
    .boolean()
    .describe(
      "The session must set a new PIN before anything but reading the session and signing out",
    ),
});

export type IdentitySignedIn = zod.input<typeof IdentitySignedIn>;
export type IdentitySignedInOutput = zod.output<typeof IdentitySignedIn>;

export const IdentitySignupLinkURL = zod.object({
  link: IdentityPendingLink,
  url: zod.url(),
});

export type IdentitySignupLinkURL = zod.input<typeof IdentitySignupLinkURL>;
export type IdentitySignupLinkURLOutput = zod.output<typeof IdentitySignupLinkURL>;

export const IndexersChanged = zod.object({
  indexerId: zod.uuid(),
});

export type IndexersChanged = zod.input<typeof IndexersChanged>;
export type IndexersChangedOutput = zod.output<typeof IndexersChanged>;

export const IndexersSearchMode = zod.object({
  mode: zod.string().describe("The Torznab t= value"),
  params: zod.array(zod.string()).describe("In the Indexer's order"),
});

export type IndexersSearchMode = zod.input<typeof IndexersSearchMode>;
export type IndexersSearchModeOutput = zod.output<typeof IndexersSearchMode>;

export const IndexersIndexer = zod.object({
  apiKeySet: zod
    .boolean()
    .describe("False when the key was dropped and an Admin must enter it again"),
  createdAt: zod.iso.datetime({ offset: true }),
  homeNodeId: zod.uuid().describe("The Node that reaches the Indexer"),
  id: zod.uuid(),
  name: zod.string(),
  protocol: zod.enum(["torznab"]),
  searchModes: zod.array(IndexersSearchMode).describe("By mode"),
  url: zod.string(),
});

export type IndexersIndexer = zod.input<typeof IndexersIndexer>;
export type IndexersIndexerOutput = zod.output<typeof IndexersIndexer>;

export const IndexersIndexerList = zod.object({
  indexers: zod.array(IndexersIndexer).describe("By name"),
});

export type IndexersIndexerList = zod.input<typeof IndexersIndexerList>;
export type IndexersIndexerListOutput = zod.output<typeof IndexersIndexerList>;

export const indexersNewAPIKeyApiKeyMax = 256;

export const IndexersNewAPIKey = zod.object({
  apiKey: zod.string().max(indexersNewAPIKeyApiKeyMax),
});

export type IndexersNewAPIKey = zod.input<typeof IndexersNewAPIKey>;
export type IndexersNewAPIKeyOutput = zod.output<typeof IndexersNewAPIKey>;

export const indexersNewIndexerApiKeyMax = 256;

export const indexersNewIndexerNameMax = 100;

export const indexersNewIndexerUrlMax = 2048;

export const indexersNewIndexerUrlRegExp = new RegExp("^https?://[^/?#]+(/[^?#]*)?$");

export const IndexersNewIndexer = zod.object({
  apiKey: zod.string().max(indexersNewIndexerApiKeyMax),
  name: zod.string().min(1).max(indexersNewIndexerNameMax),
  url: zod
    .string()
    .max(indexersNewIndexerUrlMax)
    .regex(indexersNewIndexerUrlRegExp)
    .describe("The Torznab feed's base; SlipStream adds /api"),
});

export type IndexersNewIndexer = zod.input<typeof IndexersNewIndexer>;
export type IndexersNewIndexerOutput = zod.output<typeof IndexersNewIndexer>;

export const joinInputBodyDisplayNameMax = 100;

export const joinInputBodyPinRegExp = new RegExp("^[0-9]{4}$");

export const JoinInputBody = zod.object({
  displayName: zod.string().min(1).max(joinInputBodyDisplayNameMax),
  pin: zod.string().regex(joinInputBodyPinRegExp),
});

export type JoinInputBody = zod.input<typeof JoinInputBody>;
export type JoinInputBodyOutput = zod.output<typeof JoinInputBody>;

export const LiveAccessChanged = zod.object({});

export type LiveAccessChanged = zod.input<typeof LiveAccessChanged>;
export type LiveAccessChangedOutput = zod.output<typeof LiveAccessChanged>;

export const LiveResync = zod.object({});

export type LiveResync = zod.input<typeof LiveResync>;
export type LiveResyncOutput = zod.output<typeof LiveResync>;

export const LiveSessionEnded = zod.object({
  reason: zod.enum(["signed-out", "expired", "revoked"]),
});

export type LiveSessionEnded = zod.input<typeof LiveSessionEnded>;
export type LiveSessionEndedOutput = zod.output<typeof LiveSessionEnded>;

export const MailSettings = zod.object({
  fromAddress: zod.string(),
  fromName: zod.string(),
  host: zod.string().describe("Empty until email is set up"),
  passwordSet: zod.boolean().describe("Whether a password is saved; it is never shown"),
  port: zod.int().min(1),
  security: zod.enum(["none", "starttls", "tls"]),
  username: zod.string().describe("Empty when the server takes no sign-in"),
});

export type MailSettings = zod.input<typeof MailSettings>;
export type MailSettingsOutput = zod.output<typeof MailSettings>;

export const mailSettingsPatchFromAddressMax = 254;

export const mailSettingsPatchFromNameMax = 100;

export const mailSettingsPatchHostMax = 253;

export const mailSettingsPatchPasswordMax = 1024;

export const mailSettingsPatchUsernameMax = 254;

export const MailSettingsPatch = zod.object({
  fromAddress: zod.string().max(mailSettingsPatchFromAddressMax).exactOptional(),
  fromName: zod.string().max(mailSettingsPatchFromNameMax).exactOptional(),
  host: zod.string().max(mailSettingsPatchHostMax).exactOptional(),
  password: zod
    .string()
    .max(mailSettingsPatchPasswordMax)
    .exactOptional()
    .describe("Write-only; empty clears it"),
  port: zod.int().min(1).exactOptional(),
  security: zod.enum(["none", "starttls", "tls"]).exactOptional(),
  username: zod.string().max(mailSettingsPatchUsernameMax).exactOptional(),
});

export type MailSettingsPatch = zod.input<typeof MailSettingsPatch>;
export type MailSettingsPatchOutput = zod.output<typeof MailSettingsPatch>;

export const newPINInputBodyPinRegExp = new RegExp("^[0-9]{4}$");

export const NewPINInputBody = zod.object({
  pin: zod.string().regex(newPINInputBodyPinRegExp),
});

export type NewPINInputBody = zod.input<typeof NewPINInputBody>;
export type NewPINInputBodyOutput = zod.output<typeof NewPINInputBody>;

export const newSignupLinkInputBodyEmailMax = 254;

export const NewSignupLinkInputBody = zod.object({
  email: zod.email().max(newSignupLinkInputBodyEmailMax),
});

export type NewSignupLinkInputBody = zod.input<typeof NewSignupLinkInputBody>;
export type NewSignupLinkInputBodyOutput = zod.output<typeof NewSignupLinkInputBody>;

export const NodesBuild = zod.object({
  arch: zod.string(),
  installKind: zod.string(),
  os: zod.string(),
  version: zod.string(),
});

export type NodesBuild = zod.input<typeof NodesBuild>;
export type NodesBuildOutput = zod.output<typeof NodesBuild>;

export const NodesChanged = zod.object({
  nodeId: zod.uuid(),
});

export type NodesChanged = zod.input<typeof NodesChanged>;
export type NodesChangedOutput = zod.output<typeof NodesChanged>;

export const NodesFacts = zod.object({
  appliedVersion: zod.string(),
  assigned: zod.array(zod.string()),
  httpAddress: zod
    .string()
    .describe("The address the Node's web server bound; empty until it binds"),
  restarting: zod.array(zod.string()),
  running: zod.array(zod.string()),
});

export type NodesFacts = zod.input<typeof NodesFacts>;
export type NodesFactsOutput = zod.output<typeof NodesFacts>;

export const NodesJoinCode = zod.object({
  code: zod.string(),
  expiresAt: zod.iso.datetime({ offset: true }),
});

export type NodesJoinCode = zod.input<typeof NodesJoinCode>;
export type NodesJoinCodeOutput = zod.output<typeof NodesJoinCode>;

export const NodesLink = zod.object({
  build: NodesBuild.exactOptional().describe("Absent while the link is down"),
  since: zod.iso.datetime({ offset: true }),
  state: zod.enum(["local", "up", "down", "mismatched"]),
});

export type NodesLink = zod.input<typeof NodesLink>;
export type NodesLinkOutput = zod.output<typeof NodesLink>;

export const nodesNodeIntentVersionRegExp = new RegExp("^[0-9a-f]{32}$");

export const NodesNode = zod.object({
  addresses: zod.array(zod.string()),
  facts: NodesFacts.exactOptional().describe("Absent while the Node has no session"),
  id: zod.uuid(),
  intentVersion: zod.string().regex(nodesNodeIntentVersionRegExp),
  key: zod.string().describe("The key's fingerprint"),
  link: NodesLink,
  name: zod.string(),
  pairedAt: zod.iso.datetime({ offset: true }),
  roles: zod.array(
    zod.enum([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ]),
  ),
  self: zod.boolean(),
});

export type NodesNode = zod.input<typeof NodesNode>;
export type NodesNodeOutput = zod.output<typeof NodesNode>;

export const NodesNodeList = zod.object({
  nodes: zod.array(NodesNode),
});

export type NodesNodeList = zod.input<typeof NodesNodeList>;
export type NodesNodeListOutput = zod.output<typeof NodesNodeList>;

export const NodesRoles = zod.object({
  roles: zod.array(
    zod.enum([
      "searching",
      "downloading",
      "storage",
      "converting",
      "subtitles",
      "metadata",
      "media-server-link",
      "notifying",
      "relaying",
      "serving",
    ]),
  ),
});

export type NodesRoles = zod.input<typeof NodesRoles>;
export type NodesRolesOutput = zod.output<typeof NodesRoles>;

export const NodesSettings = zod.object({
  joinCodeLifetime: SettingDurationField,
});

export type NodesSettings = zod.input<typeof NodesSettings>;
export type NodesSettingsOutput = zod.output<typeof NodesSettings>;

export const NodesSettingsChanged = zod.object({
  nodeId: zod.uuid(),
});

export type NodesSettingsChanged = zod.input<typeof NodesSettingsChanged>;
export type NodesSettingsChangedOutput = zod.output<typeof NodesSettingsChanged>;

export const NodesSettingsPatch = zod.object({
  joinCodeLifetime: zod.int().nullish().describe("Seconds; null resets it to its default"),
});

export type NodesSettingsPatch = zod.input<typeof NodesSettingsPatch>;
export type NodesSettingsPatchOutput = zod.output<typeof NodesSettingsPatch>;

export const ownPatchInputBodyDisplayNameMax = 100;

export const OwnPatchInputBody = zod.object({
  displayName: zod.string().min(1).max(ownPatchInputBodyDisplayNameMax).exactOptional(),
  keyPrompt: zod
    .enum(["ask", "never"])
    .exactOptional()
    .describe("Whether the prompt to add a passkey may show after sign-in"),
});

export type OwnPatchInputBody = zod.input<typeof OwnPatchInputBody>;
export type OwnPatchInputBodyOutput = zod.output<typeof OwnPatchInputBody>;

export const PairingController = zod.object({
  key: zod.string().describe("The Controller key's fingerprint"),
  link: zod.string(),
  name: zod.string(),
});

export type PairingController = zod.input<typeof PairingController>;
export type PairingControllerOutput = zod.output<typeof PairingController>;

export const pairingJoinBodyCodeMax = 1024;

export const PairingJoinBody = zod.object({
  code: zod.string().min(1).max(pairingJoinBodyCodeMax),
});

export type PairingJoinBody = zod.input<typeof PairingJoinBody>;
export type PairingJoinBodyOutput = zod.output<typeof PairingJoinBody>;

export const PairingWorker = zod.object({
  controller: PairingController,
  nodeId: zod.uuid(),
});

export type PairingWorker = zod.input<typeof PairingWorker>;
export type PairingWorkerOutput = zod.output<typeof PairingWorker>;

export const passkeyRenameInputBodyNameMax = 100;

export const PasskeyRenameInputBody = zod.object({
  name: zod.string().min(1).max(passkeyRenameInputBodyNameMax),
});

export type PasskeyRenameInputBody = zod.input<typeof PasskeyRenameInputBody>;
export type PasskeyRenameInputBodyOutput = zod.output<typeof PasskeyRenameInputBody>;

export const Problem = zod.object({
  code: zod.string(),
  detail: zod.string().exactOptional(),
  errors: zod.array(FieldError).exactOptional(),
  status: zod.int(),
  title: zod.string(),
});

export type Problem = zod.input<typeof Problem>;
export type ProblemOutput = zod.output<typeof Problem>;

export const ResetLinkInputBody = zod.object({
  removePasskeys: zod
    .boolean()
    .exactOptional()
    .describe("Also remove the Account's passkeys, for when someone else may have had access"),
});

export type ResetLinkInputBody = zod.input<typeof ResetLinkInputBody>;
export type ResetLinkInputBodyOutput = zod.output<typeof ResetLinkInputBody>;

export const RootsChanged = zod.object({});

export type RootsChanged = zod.input<typeof RootsChanged>;
export type RootsChangedOutput = zod.output<typeof RootsChanged>;

export const rootsNewRootContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");
export const rootsNewRootPathMax = 4096;

export const RootsNewRoot = zod.object({
  contentType: zod.string().regex(rootsNewRootContentTypeRegExp),
  path: zod
    .string()
    .min(1)
    .max(rootsNewRootPathMax)
    .describe("An absolute path to an existing folder on the Controller"),
});

export type RootsNewRoot = zod.input<typeof RootsNewRoot>;
export type RootsNewRootOutput = zod.output<typeof RootsNewRoot>;

export const rootsRootContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const RootsRoot = zod.object({
  contentType: zod.string().regex(rootsRootContentTypeRegExp),
  id: zod.uuid(),
  nodeId: zod.uuid().describe("The Node whose disk holds it"),
  path: zod.string(),
});

export type RootsRoot = zod.input<typeof RootsRoot>;
export type RootsRootOutput = zod.output<typeof RootsRoot>;

export const RootsRootList = zod.object({
  roots: zod.array(RootsRoot).describe("By Content Type"),
});

export type RootsRootList = zod.input<typeof RootsRootList>;
export type RootsRootListOutput = zod.output<typeof RootsRootList>;

export const SettingChoiceOption = zod.object({
  label: zod.string(),
  value: zod.string(),
});

export type SettingChoiceOption = zod.input<typeof SettingChoiceOption>;
export type SettingChoiceOptionOutput = zod.output<typeof SettingChoiceOption>;

export const SettingChoiceField = zod.object({
  applies: zod.enum(["live", "restart"]),
  default: zod.string(),
  defaultFrom: zod
    .string()
    .exactOptional()
    .describe("Where the default comes from, when it is another setting's value"),
  isDefault: zod
    .boolean()
    .describe("The setting has no override; Reset to default leaves it unchanged"),
  key: zod.string(),
  kind: zod.enum(["choice"]),
  label: zod.string(),
  lockedBy: zod
    .string()
    .exactOptional()
    .describe("The environment variable that sets the value; the control is read-only"),
  options: zod.array(SettingChoiceOption),
  pendingRestart: zod
    .boolean()
    .describe("The saved value differs from the running one until the Node restarts"),
  running: zod.string(),
  source: zod.enum(["database", "default", "config.yaml", ".env", "env"]),
  value: zod.string(),
});

export type SettingChoiceField = zod.input<typeof SettingChoiceField>;
export type SettingChoiceFieldOutput = zod.output<typeof SettingChoiceField>;

export const SettingTimes = zod.object({
  end: zod.string().describe("HH:MM, before start when the window crosses midnight"),
  start: zod.string().describe("HH:MM"),
});

export type SettingTimes = zod.input<typeof SettingTimes>;
export type SettingTimesOutput = zod.output<typeof SettingTimes>;

export const SettingWindowField = zod.object({
  applies: zod.enum(["live", "restart"]),
  default: SettingTimes,
  defaultFrom: zod
    .string()
    .exactOptional()
    .describe("Where the default comes from, when it is another setting's value"),
  isDefault: zod
    .boolean()
    .describe("The setting has no override; Reset to default leaves it unchanged"),
  key: zod.string(),
  kind: zod.enum(["window"]),
  label: zod.string(),
  lockedBy: zod
    .string()
    .exactOptional()
    .describe("The environment variable that sets the value; the control is read-only"),
  pendingRestart: zod
    .boolean()
    .describe("The saved value differs from the running one until the Node restarts"),
  running: SettingTimes,
  source: zod.enum(["database", "default", "config.yaml", ".env", "env"]),
  value: SettingTimes,
  zone: zod.string().describe("The time zone's name, such as Europe/London"),
});

export type SettingWindowField = zod.input<typeof SettingWindowField>;
export type SettingWindowFieldOutput = zod.output<typeof SettingWindowField>;

export const SettingPortField = zod.object({
  applies: zod.enum(["live", "restart"]),
  default: zod.int(),
  defaultFrom: zod
    .string()
    .exactOptional()
    .describe("Where the default comes from, when it is another setting's value"),
  isDefault: zod
    .boolean()
    .describe("The setting has no override; Reset to default leaves it unchanged"),
  key: zod.string(),
  kind: zod.enum(["port"]),
  label: zod.string(),
  lockedBy: zod
    .string()
    .exactOptional()
    .describe("The environment variable that sets the value; the control is read-only"),
  pendingRestart: zod
    .boolean()
    .describe("The saved value differs from the running one until the Node restarts"),
  running: zod.int(),
  source: zod.enum(["database", "default", "config.yaml", ".env", "env"]),
  value: zod.int(),
});

export type SettingPortField = zod.input<typeof SettingPortField>;
export type SettingPortFieldOutput = zod.output<typeof SettingPortField>;

export const SettingSwitchField = zod.object({
  applies: zod.enum(["live", "restart"]),
  default: zod.boolean(),
  defaultFrom: zod
    .string()
    .exactOptional()
    .describe("Where the default comes from, when it is another setting's value"),
  isDefault: zod
    .boolean()
    .describe("The setting has no override; Reset to default leaves it unchanged"),
  key: zod.string(),
  kind: zod.enum(["switch"]),
  label: zod.string(),
  lockedBy: zod
    .string()
    .exactOptional()
    .describe("The environment variable that sets the value; the control is read-only"),
  pendingRestart: zod
    .boolean()
    .describe("The saved value differs from the running one until the Node restarts"),
  running: zod.boolean(),
  source: zod.enum(["database", "default", "config.yaml", ".env", "env"]),
  value: zod.boolean(),
});

export type SettingSwitchField = zod.input<typeof SettingSwitchField>;
export type SettingSwitchFieldOutput = zod.output<typeof SettingSwitchField>;

export const SettingField = zod.discriminatedUnion("kind", [
  SettingDurationField,
  SettingCountField,
  SettingChoiceField,
  SettingWindowField,
  SettingPortField,
  SettingSwitchField,
]);

export type SettingField = zod.input<typeof SettingField>;
export type SettingFieldOutput = zod.output<typeof SettingField>;

export const shellCountsActivityMin = 0;

export const ShellCounts = zod.object({
  activity: zod.int().min(shellCountsActivityMin).describe("The Grabs neither imported nor failed"),
});

export type ShellCounts = zod.input<typeof ShellCounts>;
export type ShellCountsOutput = zod.output<typeof ShellCounts>;

export const SystemBuild = zod.object({
  commit: zod.string().exactOptional(),
  fingerprint: zod.string().exactOptional(),
  libtorrent: zod.string().exactOptional(),
  nativeBundle: zod.string().exactOptional(),
  stamped: zod.boolean(),
  target: zod.string().exactOptional(),
  version: zod.string().exactOptional(),
});

export type SystemBuild = zod.input<typeof SystemBuild>;
export type SystemBuildOutput = zod.output<typeof SystemBuild>;

export const SystemConfigSection = zod.object({
  fields: zod.array(SettingField),
  id: zod.enum(["web", "log"]),
});

export type SystemConfigSection = zod.input<typeof SystemConfigSection>;
export type SystemConfigSectionOutput = zod.output<typeof SystemConfigSection>;

export const SystemHealth = zod.object({
  status: zod.enum(["ok"]),
});

export type SystemHealth = zod.input<typeof SystemHealth>;
export type SystemHealthOutput = zod.output<typeof SystemHealth>;

export const SystemHttpFacts = zod.object({
  configured: zod
    .string()
    .describe("http.listen at start; listeners.http differs after a fallback to a later port"),
});

export type SystemHttpFacts = zod.input<typeof SystemHttpFacts>;
export type SystemHttpFactsOutput = zod.output<typeof SystemHttpFacts>;

export const SystemListeners = zod.object({
  control: zod.string(),
  http: zod.string(),
  link: zod.string(),
  torrent: zod.string(),
});

export type SystemListeners = zod.input<typeof SystemListeners>;
export type SystemListenersOutput = zod.output<typeof SystemListeners>;

export const SystemLogRecord = zod.object({
  attrs: zod.unknown().describe("The record's other attributes"),
  level: zod.enum(["debug", "info", "warn", "error"]),
  msg: zod.string(),
  seq: zod.int(),
  subsystem: zod.string(),
  time: zod.iso.datetime({ offset: true }),
});

export type SystemLogRecord = zod.input<typeof SystemLogRecord>;
export type SystemLogRecordOutput = zod.output<typeof SystemLogRecord>;

export const SystemLogReset = zod.object({
  reason: zod.enum(["lapped", "restarted"]),
  startedAt: zod.iso.datetime({ offset: true }).describe("When the serving process started"),
});

export type SystemLogReset = zod.input<typeof SystemLogReset>;
export type SystemLogResetOutput = zod.output<typeof SystemLogReset>;

export const SystemNodeConfig = zod.object({
  sections: zod.array(SystemConfigSection),
});

export type SystemNodeConfig = zod.input<typeof SystemNodeConfig>;
export type SystemNodeConfigOutput = zod.output<typeof SystemNodeConfig>;

export const SystemPasskeys = zod.object({
  available: zod.boolean(),
  reason: zod
    .enum(["ip_address", "plain_http", "not_a_domain"])
    .exactOptional()
    .describe(
      "Why it can't: an IP address, a name over plain HTTP other than localhost, or a host that is not a domain name",
    ),
});

export type SystemPasskeys = zod.input<typeof SystemPasskeys>;
export type SystemPasskeysOutput = zod.output<typeof SystemPasskeys>;

export const SystemReport = zod.object({
  build: SystemBuild,
  dataDir: zod.string(),
  http: SystemHttpFacts,
  listeners: SystemListeners,
  logsDir: zod.string(),
  pid: zod.int(),
  seed: zod.string(),
  startedAt: zod.iso.datetime({ offset: true }),
});

export type SystemReport = zod.input<typeof SystemReport>;
export type SystemReportOutput = zod.output<typeof SystemReport>;

export const SystemRestarting = zod.object({
  boundPort: zod.int().describe("The port the stopping process serves on"),
  nextPort: zod
    .int()
    .exactOptional()
    .describe("The port the successor binds, when http.listen waits for this restart"),
  startedAt: zod.iso
    .datetime({ offset: true })
    .describe("When the stopping process started; its successor's startedAt is later"),
});

export type SystemRestarting = zod.input<typeof SystemRestarting>;
export type SystemRestartingOutput = zod.output<typeof SystemRestarting>;

export const SystemStatus = zod.object({
  claim: zod.enum(["unclaimed", "controller", "worker"]),
  passkeys: SystemPasskeys.describe(
    "Whether the address this request reached SlipStream at can hold a passkey",
  ),
});

export type SystemStatus = zod.input<typeof SystemStatus>;
export type SystemStatusOutput = zod.output<typeof SystemStatus>;

export const TargetsChanged = zod.object({});

export type TargetsChanged = zod.input<typeof TargetsChanged>;
export type TargetsChangedOutput = zod.output<typeof TargetsChanged>;

export const targetsNewProfileContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");
export const targetsNewProfileNameMax = 100;

export const TargetsNewProfile = zod.object({
  contentType: zod.string().regex(targetsNewProfileContentTypeRegExp),
  name: zod.string().min(1).max(targetsNewProfileNameMax),
  targetIds: zod.array(zod.uuid()).min(1).describe("Highest ranked first"),
});

export type TargetsNewProfile = zod.input<typeof TargetsNewProfile>;
export type TargetsNewProfileOutput = zod.output<typeof TargetsNewProfile>;

export const targetsNewTargetNameMax = 100;

export const TargetsNewTarget = zod.object({
  name: zod.string().min(1).max(targetsNewTargetNameMax),
  resolutions: zod
    .array(zod.enum(["2160p", "1080p", "720p", "576p", "480p"]))
    .min(1)
    .describe("Highest ranked first"),
});

export type TargetsNewTarget = zod.input<typeof TargetsNewTarget>;
export type TargetsNewTargetOutput = zod.output<typeof TargetsNewTarget>;

export const targetsProfileContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const TargetsProfile = zod.object({
  contentType: zod.string().regex(targetsProfileContentTypeRegExp),
  default: zod.boolean().describe("Whether the Content Type's Works use it"),
  id: zod.uuid(),
  name: zod.string(),
  targetIds: zod.array(zod.uuid()).describe("Highest ranked first"),
});

export type TargetsProfile = zod.input<typeof TargetsProfile>;
export type TargetsProfileOutput = zod.output<typeof TargetsProfile>;

export const TargetsProfileList = zod.object({
  profiles: zod.array(TargetsProfile).describe("By Content Type, then name"),
});

export type TargetsProfileList = zod.input<typeof TargetsProfileList>;
export type TargetsProfileListOutput = zod.output<typeof TargetsProfileList>;

export const TargetsTarget = zod.object({
  createdAt: zod.iso.datetime({ offset: true }),
  id: zod.uuid(),
  name: zod.string(),
  resolutions: zod
    .array(zod.enum(["2160p", "1080p", "720p", "576p", "480p"]))
    .describe("Highest ranked first"),
});

export type TargetsTarget = zod.input<typeof TargetsTarget>;
export type TargetsTargetOutput = zod.output<typeof TargetsTarget>;

export const TargetsTargetList = zod.object({
  targets: zod.array(TargetsTarget).describe("By name"),
});

export type TargetsTargetList = zod.input<typeof TargetsTargetList>;
export type TargetsTargetListOutput = zod.output<typeof TargetsTargetList>;

export const TasksCadence = zod.object({
  field: SettingDurationField,
  inWindow: zod
    .boolean()
    .describe("The task starts in the Maintenance window, every field.value seconds in whole days"),
  words: zod.string().describe("The cadence as the Tasks page says it, such as Every 8 h"),
});

export type TasksCadence = zod.input<typeof TasksCadence>;
export type TasksCadenceOutput = zod.output<typeof TasksCadence>;

export const TasksChanged = zod.object({
  taskId: zod.string(),
});

export type TasksChanged = zod.input<typeof TasksChanged>;
export type TasksChangedOutput = zod.output<typeof TasksChanged>;

export const TasksRunUnit = zod.object({
  cause: zod.string().nullable().describe("Why it failed or was skipped, null when it succeeded"),
  key: zod.string(),
  label: zod.string(),
  node: zod.string().describe("Empty when the Node is no longer paired"),
  outcome: zod.enum(["succeeded", "failed", "skipped", "canceled"]),
});

export type TasksRunUnit = zod.input<typeof TasksRunUnit>;
export type TasksRunUnitOutput = zod.output<typeof TasksRunUnit>;

export const TasksRun = zod.object({
  cause: zod.string().nullable().describe("The first failure's cause, null when none"),
  endedAt: zod.iso.datetime({ offset: true }).nullable().describe("Null while it runs"),
  id: zod.int(),
  nodes: zod.array(zod.string()).describe("The Nodes its units ran on"),
  result: zod
    .union([
      zod.literal("succeeded"),
      zod.literal("partial"),
      zod.literal("failed"),
      zod.literal("skipped"),
      zod.literal("canceled"),
      zod.literal(null),
    ])
    .nullable()
    .describe("Null while it runs"),
  startedAt: zod.iso.datetime({ offset: true }),
  summary: zod.string().describe("Such as 5 of 6 steps, seedbox offline; empty while it runs"),
  trigger: zod.enum(["schedule", "catch-up", "run-now", "returned"]),
  units: zod
    .array(TasksRunUnit)
    .describe(
      "Each unit's outcome as the run ended, kept after the unit is gone; empty while it runs",
    ),
});

export type TasksRun = zod.input<typeof TasksRun>;
export type TasksRunOutput = zod.output<typeof TasksRun>;

export const TasksRunNow = zod.object({
  available: zod.boolean(),
  reason: zod
    .enum(["running", "waiting"])
    .exactOptional()
    .describe("Why Run now is unavailable; absent when it is available"),
});

export type TasksRunNow = zod.input<typeof TasksRunNow>;
export type TasksRunNowOutput = zod.output<typeof TasksRunNow>;

export const TasksIdleState = zod.object({
  kind: zod.enum(["idle"]),
  next: zod.iso
    .datetime({ offset: true })
    .describe("When the next run is due; now when it is due already"),
});

export type TasksIdleState = zod.input<typeof TasksIdleState>;
export type TasksIdleStateOutput = zod.output<typeof TasksIdleState>;

export const tasksRunningStateDoneMin = 0;

export const tasksRunningStateTotalMin = 0;

export const TasksRunningState = zod.object({
  done: zod.int().min(tasksRunningStateDoneMin),
  kind: zod.enum(["running"]),
  runId: zod.int(),
  since: zod.iso.datetime({ offset: true }),
  total: zod.int().min(tasksRunningStateTotalMin),
  trigger: zod.enum(["schedule", "catch-up", "run-now", "returned"]),
});

export type TasksRunningState = zod.input<typeof TasksRunningState>;
export type TasksRunningStateOutput = zod.output<typeof TasksRunningState>;

export const TasksWaitingState = zod.object({
  kind: zod.enum(["waiting"]),
  link: zod.enum(["down", "mismatched"]),
  node: zod.string().describe("The Node every unit waits for"),
  since: zod.iso.datetime({ offset: true }).describe("When the Node's link entered that state"),
});

export type TasksWaitingState = zod.input<typeof TasksWaitingState>;
export type TasksWaitingStateOutput = zod.output<typeof TasksWaitingState>;

export const TasksState = zod.discriminatedUnion("kind", [
  TasksIdleState,
  TasksRunningState,
  TasksWaitingState,
]);

export type TasksState = zod.input<typeof TasksState>;
export type TasksStateOutput = zod.output<typeof TasksState>;

export const TasksTask = zod.object({
  about: zod.string(),
  cadence: TasksCadence,
  group: zod.enum(["controller", "acquire", "store", "process", "connect", "reach"]),
  id: zod.string(),
  lastRun: TasksRun.exactOptional().describe("The latest finished run, absent before one has"),
  name: zod.string(),
  nodes: zod.array(zod.string()).describe("The Nodes its units run on"),
  noun: SettingNoun.describe("What its progress counts: 3 of 12 steps"),
  overdue: SettingChoiceField,
  runNow: TasksRunNow,
  state: TasksState,
});

export type TasksTask = zod.input<typeof TasksTask>;
export type TasksTaskOutput = zod.output<typeof TasksTask>;

export const TasksUnit = zod.object({
  cause: zod.string().nullable().describe("Why it last failed or was skipped, null otherwise"),
  dueOnReturn: zod.boolean().describe("It runs once when its Node returns"),
  endedAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When it last ended, null before it has run"),
  key: zod.string(),
  label: zod.string(),
  lastSuccessAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("Null before it has succeeded"),
  node: zod.string().describe("The Node it runs on"),
  outcome: zod
    .union([
      zod.literal("succeeded"),
      zod.literal("failed"),
      zod.literal("skipped"),
      zod.literal("canceled"),
      zod.literal(null),
    ])
    .nullable()
    .describe("Its last outcome, null before it has run"),
});

export type TasksUnit = zod.input<typeof TasksUnit>;
export type TasksUnitOutput = zod.output<typeof TasksUnit>;

export const TasksDetail = zod.object({
  runs: zod.array(TasksRun).describe("Its runs, newest first, up to the runs kept"),
  task: TasksTask,
  units: zod.array(TasksUnit).describe("Each unit it lists now, in run order"),
});

export type TasksDetail = zod.input<typeof TasksDetail>;
export type TasksDetailOutput = zod.output<typeof TasksDetail>;

export const TasksFailurePart = zod.object({
  href: zod.string().describe("The task's page"),
  kind: zod.enum(["failure"]),
  text: zod.string(),
});

export type TasksFailurePart = zod.input<typeof TasksFailurePart>;
export type TasksFailurePartOutput = zod.output<typeof TasksFailurePart>;

export const TasksGroup = zod.object({
  group: zod.enum(["controller", "acquire", "store", "process", "connect", "reach"]),
  tasks: zod.array(TasksTask).min(1),
});

export type TasksGroup = zod.input<typeof TasksGroup>;
export type TasksGroupOutput = zod.output<typeof TasksGroup>;

export const tasksOpenRunDoneMin = 0;

export const tasksOpenRunTotalMin = 0;

export const TasksOpenRun = zod.object({
  done: zod.int().min(tasksOpenRunDoneMin),
  name: zod.string().describe("The task's name"),
  nodes: zod.array(zod.string()).describe("The Nodes its units run on"),
  noun: SettingNoun.describe("What its progress counts: 3 of 12 steps"),
  runId: zod.int(),
  startedAt: zod.iso.datetime({ offset: true }),
  taskId: zod.string(),
  total: zod.int().min(tasksOpenRunTotalMin),
  trigger: zod.enum(["schedule", "catch-up", "run-now", "returned"]),
});

export type TasksOpenRun = zod.input<typeof TasksOpenRun>;
export type TasksOpenRunOutput = zod.output<typeof TasksOpenRun>;

export const TasksOpenRuns = zod.object({
  headerDelaySeconds: zod
    .int()
    .describe(
      "How long a scheduled run is open before the header shows it; a run-now run shows at once",
    ),
  runs: zod.array(TasksOpenRun).describe("Oldest first"),
});

export type TasksOpenRuns = zod.input<typeof TasksOpenRuns>;
export type TasksOpenRunsOutput = zod.output<typeof TasksOpenRuns>;

export const TasksPlainPart = zod.object({
  kind: zod.enum(["plain"]),
  text: zod.string(),
});

export type TasksPlainPart = zod.input<typeof TasksPlainPart>;
export type TasksPlainPartOutput = zod.output<typeof TasksPlainPart>;

export const TasksWarningPart = zod.object({
  href: zod.string().describe("The task's page"),
  kind: zod.enum(["warning"]),
  text: zod.string(),
});

export type TasksWarningPart = zod.input<typeof TasksWarningPart>;
export type TasksWarningPartOutput = zod.output<typeof TasksWarningPart>;

export const TasksPart = zod.discriminatedUnion("kind", [
  TasksPlainPart,
  TasksWarningPart,
  TasksFailurePart,
]);

export type TasksPart = zod.input<typeof TasksPart>;
export type TasksPartOutput = zod.output<typeof TasksPart>;

export const TasksSchedule = zod.object({
  headerDelay: SettingDurationField,
  jitterCap: SettingDurationField,
  jitterPercent: SettingCountField,
  overdue: SettingChoiceField,
  partialWarnIntervals: SettingCountField,
  runsKept: SettingCountField,
  window: SettingWindowField,
});

export type TasksSchedule = zod.input<typeof TasksSchedule>;
export type TasksScheduleOutput = zod.output<typeof TasksSchedule>;

export const TasksOverview = zod.object({
  groups: zod.array(TasksGroup).describe("Each group with a shown task, in the page's fixed order"),
  multinode: zod
    .boolean()
    .describe("The install has more than one Node, so the Tasks page shows a Node column"),
  sentence: zod.array(TasksPart).min(1).describe("The status sentence's parts, in order"),
  settings: TasksSchedule,
});

export type TasksOverview = zod.input<typeof TasksOverview>;
export type TasksOverviewOutput = zod.output<typeof TasksOverview>;

export const TasksProgressed = zod.object({
  done: zod.int(),
  runId: zod.int(),
  taskId: zod.string(),
  total: zod.int(),
});

export type TasksProgressed = zod.input<typeof TasksProgressed>;
export type TasksProgressedOutput = zod.output<typeof TasksProgressed>;

export const TasksRunStarted = zod.object({
  runId: zod.int(),
});

export type TasksRunStarted = zod.input<typeof TasksRunStarted>;
export type TasksRunStartedOutput = zod.output<typeof TasksRunStarted>;

export const TasksSchedulePatch = zod.object({
  headerDelay: zod.int().nullish().describe("Seconds; null resets it to its default"),
  jitterCap: zod.int().nullish().describe("Seconds; null resets it to its default"),
  jitterPercent: zod.int().nullish().describe("null resets it to its default"),
  overdue: zod.string().nullish().describe("catch-up or wait; null resets it to its default"),
  partialWarnIntervals: zod.int().nullish().describe("null resets it to its default"),
  runsKept: zod.int().nullish().describe("null resets it to its default"),
  window: zod
    .union([SettingTimes, zod.null()])
    .exactOptional()
    .describe("null resets it to its default"),
});

export type TasksSchedulePatch = zod.input<typeof TasksSchedulePatch>;
export type TasksSchedulePatchOutput = zod.output<typeof TasksSchedulePatch>;

export const TasksTaskPatch = zod.object({
  cadence: zod.int().nullish().describe("Seconds; null resets it to its default"),
  overdue: zod.string().nullish().describe("catch-up or wait; null resets it to its default"),
});

export type TasksTaskPatch = zod.input<typeof TasksTaskPatch>;
export type TasksTaskPatchOutput = zod.output<typeof TasksTaskPatch>;

export const WorksTextValue = zod.object({
  key: zod.string(),
  kind: zod.enum(["text"]),
  text: zod.string(),
});

export type WorksTextValue = zod.input<typeof WorksTextValue>;
export type WorksTextValueOutput = zod.output<typeof WorksTextValue>;

export const WorksNumberValue = zod.object({
  key: zod.string(),
  kind: zod.enum(["number"]),
  number: zod.number(),
});

export type WorksNumberValue = zod.input<typeof WorksNumberValue>;
export type WorksNumberValueOutput = zod.output<typeof WorksNumberValue>;

export const worksDateValueDateRegExp = new RegExp("^[0-9]{4}(-[0-9]{2}(-[0-9]{2})?)?$");

export const WorksDateValue = zod.object({
  date: zod.string().regex(worksDateValueDateRegExp).describe("YYYY, YYYY-MM or YYYY-MM-DD"),
  key: zod.string(),
  kind: zod.enum(["date"]),
});

export type WorksDateValue = zod.input<typeof WorksDateValue>;
export type WorksDateValueOutput = zod.output<typeof WorksDateValue>;

export const worksDurationValueMsMin = 0;

export const WorksDurationValue = zod.object({
  key: zod.string(),
  kind: zod.enum(["duration"]),
  ms: zod.int().min(worksDurationValueMsMin),
});

export type WorksDurationValue = zod.input<typeof WorksDurationValue>;
export type WorksDurationValueOutput = zod.output<typeof WorksDurationValue>;

export const WorksFieldValue = zod.discriminatedUnion("kind", [
  WorksTextValue,
  WorksNumberValue,
  WorksDateValue,
  WorksDurationValue,
]);

export type WorksFieldValue = zod.input<typeof WorksFieldValue>;
export type WorksFieldValueOutput = zod.output<typeof WorksFieldValue>;

export const WorksItem = zod.object({
  fields: zod.array(WorksFieldValue),
  id: zod.uuid(),
  key: zod.string(),
  label: zod.string(),
  levelId: zod.uuid().nullable().describe("Null when the Content Type declares no levels"),
  monitored: zod.boolean(),
  position: zod.int().min(1),
  withdrawnAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When the source stopped listing it; null while it lists it"),
});

export type WorksItem = zod.input<typeof WorksItem>;
export type WorksItemOutput = zod.output<typeof WorksItem>;

export const WorksLevel = zod.object({
  depth: zod.int().min(1),
  fields: zod.array(WorksFieldValue),
  id: zod.uuid(),
  key: zod.string(),
  label: zod.string(),
  parentId: zod.uuid().nullable().describe("Null under the Work"),
  position: zod.int().min(1),
  withdrawnAt: zod.iso
    .datetime({ offset: true })
    .nullable()
    .describe("When the source stopped listing it; null while it lists it"),
});

export type WorksLevel = zod.input<typeof WorksLevel>;
export type WorksLevelOutput = zod.output<typeof WorksLevel>;

export const worksTreeContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const WorksTree = zod.object({
  addedAt: zod.iso.datetime({ offset: true }),
  contentType: zod.string().regex(worksTreeContentTypeRegExp),
  fields: zod.array(WorksFieldValue),
  id: zod.uuid(),
  items: zod
    .array(WorksItem)
    .describe("By their level's order, live before withdrawn, then position"),
  key: zod.string(),
  label: zod.string(),
  levels: zod
    .array(WorksLevel)
    .describe("By depth, then their parent's order, live before withdrawn, then position"),
  refreshedAt: zod.iso.datetime({ offset: true }),
  sortLabel: zod.string(),
});

export type WorksTree = zod.input<typeof WorksTree>;
export type WorksTreeOutput = zod.output<typeof WorksTree>;

export const WorksAdded = zod.object({
  created: zod.boolean().describe("False when the Work was already in the library"),
  work: WorksTree,
});

export type WorksAdded = zod.input<typeof WorksAdded>;
export type WorksAddedOutput = zod.output<typeof WorksAdded>;

export const WorksCandidate = zod.object({
  fields: zod.array(WorksFieldValue).describe("The Work's fields"),
  key: zod.string(),
  label: zod.string(),
  workId: zod.uuid().nullable().describe("The Work, once it is in the library"),
});

export type WorksCandidate = zod.input<typeof WorksCandidate>;
export type WorksCandidateOutput = zod.output<typeof WorksCandidate>;

export const WorksCandidateList = zod.object({
  candidates: zod.array(WorksCandidate).describe("Best first"),
});

export type WorksCandidateList = zod.input<typeof WorksCandidateList>;
export type WorksCandidateListOutput = zod.output<typeof WorksCandidateList>;

export const WorksFieldSpec = zod.object({
  depth: zod
    .int()
    .min(1)
    .exactOptional()
    .describe("The level's depth, 1 for the outermost; present only on tier level"),
  key: zod.string(),
  kind: zod.enum(["text", "number", "date", "duration"]),
  label: zod.string(),
  tier: zod.enum(["work", "level", "item"]),
});

export type WorksFieldSpec = zod.input<typeof WorksFieldSpec>;
export type WorksFieldSpecOutput = zod.output<typeof WorksFieldSpec>;

export const WorksNoun = zod.object({
  many: zod.string(),
  one: zod.string(),
});

export type WorksNoun = zod.input<typeof WorksNoun>;
export type WorksNounOutput = zod.output<typeof WorksNoun>;

export const worksContentTypeIdRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const WorksContentType = zod.object({
  enabled: zod.boolean(),
  fields: zod.array(WorksFieldSpec),
  icon: zod.enum(["film", "tv", "note", "open-book", "headphones", "panels", "image", "shapes"]),
  id: zod.string().regex(worksContentTypeIdRegExp),
  item: WorksNoun,
  levels: zod.array(WorksNoun).describe("Outermost first; empty when Items hang off the Work"),
  name: zod.string(),
  work: WorksNoun,
});

export type WorksContentType = zod.input<typeof WorksContentType>;
export type WorksContentTypeOutput = zod.output<typeof WorksContentType>;

export const WorksContentTypeList = zod.object({
  contentTypes: zod.array(WorksContentType),
});

export type WorksContentTypeList = zod.input<typeof WorksContentTypeList>;
export type WorksContentTypeListOutput = zod.output<typeof WorksContentTypeList>;

export const worksContentTypesChangedContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const WorksContentTypesChanged = zod.object({
  contentType: zod.string().regex(worksContentTypesChangedContentTypeRegExp),
});

export type WorksContentTypesChanged = zod.input<typeof WorksContentTypesChanged>;
export type WorksContentTypesChangedOutput = zod.output<typeof WorksContentTypesChanged>;

export const WorksEnabled = zod.object({
  enabled: zod.boolean(),
});

export type WorksEnabled = zod.input<typeof WorksEnabled>;
export type WorksEnabledOutput = zod.output<typeof WorksEnabled>;

export const worksListedWorkContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");

export const WorksListedWork = zod.object({
  addedAt: zod.iso.datetime({ offset: true }),
  contentType: zod.string().regex(worksListedWorkContentTypeRegExp),
  fields: zod.array(WorksFieldValue).describe("The Work's own fields"),
  id: zod.uuid(),
  key: zod.string(),
  label: zod.string(),
  refreshedAt: zod.iso.datetime({ offset: true }),
  sortLabel: zod.string(),
});

export type WorksListedWork = zod.input<typeof WorksListedWork>;
export type WorksListedWorkOutput = zod.output<typeof WorksListedWork>;

export const worksNewWorkContentTypeRegExp = new RegExp("^[a-z][a-z0-9]{1,15}$");
export const worksNewWorkKeyMax = 200;

export const WorksNewWork = zod.object({
  contentType: zod.string().regex(worksNewWorkContentTypeRegExp),
  key: zod
    .string()
    .min(1)
    .max(worksNewWorkKeyMax)
    .describe("The Work's key at its metadata source"),
  monitor: zod
    .boolean()
    .exactOptional()
    .describe("Monitor a new Work's Items; adding a Work already in the library changes nothing"),
});

export type WorksNewWork = zod.input<typeof WorksNewWork>;
export type WorksNewWorkOutput = zod.output<typeof WorksNewWork>;

export const WorksWorkChanged = zod.object({
  workId: zod.uuid(),
});

export type WorksWorkChanged = zod.input<typeof WorksWorkChanged>;
export type WorksWorkChangedOutput = zod.output<typeof WorksWorkChanged>;

export const WorksWorkList = zod.object({
  works: zod.array(WorksListedWork).describe("By sort label, then id"),
});

export type WorksWorkList = zod.input<typeof WorksWorkList>;
export type WorksWorkListOutput = zod.output<typeof WorksWorkList>;

export const LiveEventFilesChanged = zod.object({
  event: zod.enum(["files.changed"]),
  data: FilesChanged,
});

export type LiveEventFilesChanged = zod.input<typeof LiveEventFilesChanged>;
export type LiveEventFilesChangedOutput = zod.output<typeof LiveEventFilesChanged>;

export const LiveEventGrabsChanged = zod.object({
  event: zod.enum(["grabs.changed"]),
  data: GrabsChanged,
});

export type LiveEventGrabsChanged = zod.input<typeof LiveEventGrabsChanged>;
export type LiveEventGrabsChangedOutput = zod.output<typeof LiveEventGrabsChanged>;

export const LiveEventGrabsSearchChanged = zod.object({
  event: zod.enum(["grabs.search-changed"]),
  data: GrabsSearchChanged,
});

export type LiveEventGrabsSearchChanged = zod.input<typeof LiveEventGrabsSearchChanged>;
export type LiveEventGrabsSearchChangedOutput = zod.output<typeof LiveEventGrabsSearchChanged>;

export const LiveEventIdentityAccountsChanged = zod.object({
  event: zod.enum(["identity.accounts-changed"]),
  data: IdentityAccountsChanged,
});

export type LiveEventIdentityAccountsChanged = zod.input<typeof LiveEventIdentityAccountsChanged>;
export type LiveEventIdentityAccountsChangedOutput = zod.output<
  typeof LiveEventIdentityAccountsChanged
>;

export const LiveEventIdentityLinksChanged = zod.object({
  event: zod.enum(["identity.links-changed"]),
  data: IdentityLinksChanged,
});

export type LiveEventIdentityLinksChanged = zod.input<typeof LiveEventIdentityLinksChanged>;
export type LiveEventIdentityLinksChangedOutput = zod.output<typeof LiveEventIdentityLinksChanged>;

export const LiveEventIdentityPasskeysChanged = zod.object({
  event: zod.enum(["identity.passkeys-changed"]),
  data: IdentityPasskeysChanged,
});

export type LiveEventIdentityPasskeysChanged = zod.input<typeof LiveEventIdentityPasskeysChanged>;
export type LiveEventIdentityPasskeysChangedOutput = zod.output<
  typeof LiveEventIdentityPasskeysChanged
>;

export const LiveEventIdentitySessionsChanged = zod.object({
  event: zod.enum(["identity.sessions-changed"]),
  data: IdentitySessionsChanged,
});

export type LiveEventIdentitySessionsChanged = zod.input<typeof LiveEventIdentitySessionsChanged>;
export type LiveEventIdentitySessionsChangedOutput = zod.output<
  typeof LiveEventIdentitySessionsChanged
>;

export const LiveEventIndexersChanged = zod.object({
  event: zod.enum(["indexers.changed"]),
  data: IndexersChanged,
});

export type LiveEventIndexersChanged = zod.input<typeof LiveEventIndexersChanged>;
export type LiveEventIndexersChangedOutput = zod.output<typeof LiveEventIndexersChanged>;

export const LiveEventLiveAccessChanged = zod.object({
  event: zod.enum(["live.access-changed"]),
  data: LiveAccessChanged,
});

export type LiveEventLiveAccessChanged = zod.input<typeof LiveEventLiveAccessChanged>;
export type LiveEventLiveAccessChangedOutput = zod.output<typeof LiveEventLiveAccessChanged>;

export const LiveEventLiveResync = zod.object({
  event: zod.enum(["live.resync"]),
  data: LiveResync,
});

export type LiveEventLiveResync = zod.input<typeof LiveEventLiveResync>;
export type LiveEventLiveResyncOutput = zod.output<typeof LiveEventLiveResync>;

export const LiveEventLiveSessionEnded = zod.object({
  event: zod.enum(["live.session-ended"]),
  data: LiveSessionEnded,
});

export type LiveEventLiveSessionEnded = zod.input<typeof LiveEventLiveSessionEnded>;
export type LiveEventLiveSessionEndedOutput = zod.output<typeof LiveEventLiveSessionEnded>;

export const LiveEventNodeSettingsChanged = zod.object({
  event: zod.enum(["node.settings-changed"]),
  data: NodesSettingsChanged,
});

export type LiveEventNodeSettingsChanged = zod.input<typeof LiveEventNodeSettingsChanged>;
export type LiveEventNodeSettingsChangedOutput = zod.output<typeof LiveEventNodeSettingsChanged>;

export const LiveEventNodesChanged = zod.object({
  event: zod.enum(["nodes.changed"]),
  data: NodesChanged,
});

export type LiveEventNodesChanged = zod.input<typeof LiveEventNodesChanged>;
export type LiveEventNodesChangedOutput = zod.output<typeof LiveEventNodesChanged>;

export const LiveEventRootsChanged = zod.object({
  event: zod.enum(["roots.changed"]),
  data: RootsChanged,
});

export type LiveEventRootsChanged = zod.input<typeof LiveEventRootsChanged>;
export type LiveEventRootsChangedOutput = zod.output<typeof LiveEventRootsChanged>;

export const LiveEventTargetsChanged = zod.object({
  event: zod.enum(["targets.changed"]),
  data: TargetsChanged,
});

export type LiveEventTargetsChanged = zod.input<typeof LiveEventTargetsChanged>;
export type LiveEventTargetsChangedOutput = zod.output<typeof LiveEventTargetsChanged>;

export const LiveEventTasksChanged = zod.object({
  event: zod.enum(["tasks.changed"]),
  data: TasksChanged,
});

export type LiveEventTasksChanged = zod.input<typeof LiveEventTasksChanged>;
export type LiveEventTasksChangedOutput = zod.output<typeof LiveEventTasksChanged>;

export const LiveEventTasksProgress = zod.object({
  event: zod.enum(["tasks.progress"]),
  data: TasksProgressed,
});

export type LiveEventTasksProgress = zod.input<typeof LiveEventTasksProgress>;
export type LiveEventTasksProgressOutput = zod.output<typeof LiveEventTasksProgress>;

export const LiveEventWorksChanged = zod.object({
  event: zod.enum(["works.changed"]),
  data: WorksWorkChanged,
});

export type LiveEventWorksChanged = zod.input<typeof LiveEventWorksChanged>;
export type LiveEventWorksChangedOutput = zod.output<typeof LiveEventWorksChanged>;

export const LiveEventWorksContentTypesChanged = zod.object({
  event: zod.enum(["works.content-types-changed"]),
  data: WorksContentTypesChanged,
});

export type LiveEventWorksContentTypesChanged = zod.input<typeof LiveEventWorksContentTypesChanged>;
export type LiveEventWorksContentTypesChangedOutput = zod.output<
  typeof LiveEventWorksContentTypesChanged
>;

export const LiveEvent = zod.discriminatedUnion("event", [
  LiveEventFilesChanged,
  LiveEventGrabsChanged,
  LiveEventGrabsSearchChanged,
  LiveEventIdentityAccountsChanged,
  LiveEventIdentityLinksChanged,
  LiveEventIdentityPasskeysChanged,
  LiveEventIdentitySessionsChanged,
  LiveEventIndexersChanged,
  LiveEventLiveAccessChanged,
  LiveEventLiveResync,
  LiveEventLiveSessionEnded,
  LiveEventNodeSettingsChanged,
  LiveEventNodesChanged,
  LiveEventRootsChanged,
  LiveEventTargetsChanged,
  LiveEventTasksChanged,
  LiveEventTasksProgress,
  LiveEventWorksChanged,
  LiveEventWorksContentTypesChanged,
]);

export type LiveEvent = zod.input<typeof LiveEvent>;
export type LiveEventOutput = zod.output<typeof LiveEvent>;

export const LogEventLogRecord = zod.object({
  event: zod.enum(["log.record"]),
  data: SystemLogRecord,
});

export type LogEventLogRecord = zod.input<typeof LogEventLogRecord>;
export type LogEventLogRecordOutput = zod.output<typeof LogEventLogRecord>;

export const LogEventLogReset = zod.object({
  event: zod.enum(["log.reset"]),
  data: SystemLogReset,
});

export type LogEventLogReset = zod.input<typeof LogEventLogReset>;
export type LogEventLogResetOutput = zod.output<typeof LogEventLogReset>;

export const LogEvent = zod.discriminatedUnion("event", [LogEventLogRecord, LogEventLogReset]);

export type LogEvent = zod.input<typeof LogEvent>;
export type LogEventOutput = zod.output<typeof LogEvent>;
