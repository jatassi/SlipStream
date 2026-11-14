import * as zod from "zod";

import {
  HoldersOutputBody,
  KindsOutputBody,
  MappingsOutputBody,
  MediaserversAccountView,
  MediaserversChanges,
  MediaserversDraftBody,
  MediaserversLibraries,
  MediaserversMappingsBody,
  MediaserversPatchBody,
  MediaserversPinBody,
  MediaserversPlexPin,
  MediaserversPlexPinState,
  MediaserversServerView,
  MediaserversSessions,
  MediaserversSignInBody,
  MediaserversSignedIn,
  MediaserversTested,
  PlexServersOutputBody,
  ServersOutputBody,
} from "../slipStream.schemas.zod.ts";

/**
 * @summary List the Media Servers SlipStream keeps up to date, each with its state
 */
export const ListMediaServersResponse = ServersOutputBody;

/**
 * @summary Add a Media Server once it responds from its reaching Node and accepts SlipStream's sign-in
 */
export const AddMediaServerBody = MediaserversDraftBody;

export const AddMediaServerResponse = MediaserversServerView;

/**
 * @summary List the kinds of Media Server SlipStream supports, with the sign-in each takes and its oldest supported version
 */
export const ListMediaServerKindsResponse = KindsOutputBody;

/**
 * @summary Sign out of Plex, which also removes every Plex server
 */
export const SignOutOfPlexResponse = zod.void();

/**
 * @summary Get the Plex account SlipStream is signed in to and when its sign-in expires
 */
export const GetPlexAccountResponse = MediaserversAccountView;

/**
 * @summary Start a Plex sign-in and get the plex.tv page the Admin signs in on
 */
export const StartPlexSignInBody = MediaserversPinBody;

export const StartPlexSignInResponse = MediaserversPlexPin;

/**
 * @summary Get whether a Plex sign-in is still waiting, has expired or signed in
 */
export const GetPlexSignInParams = zod.object({
  id: zod.int(),
});

export const GetPlexSignInResponse = MediaserversPlexPinState;

/**
 * @summary List the Plex account's servers and which Nodes reach each of their connections
 */
export const ListPlexServersResponse = PlexServersOutputBody;

/**
 * @summary Sign in to a Jellyfin or Emby server as an administrator once, creating an API key named SlipStream for a draft server
 */
export const SignInToMediaServerBody = MediaserversSignInBody;

export const SignInToMediaServerResponse = MediaserversSignedIn;

/**
 * @summary Remove a Media Server, which also deletes its queued changes, its path index and its play history
 */
export const RemoveMediaServerParams = zod.object({
  id: zod.uuid(),
});

export const RemoveMediaServerResponse = zod.void();

/**
 * @summary Get a Media Server with its state
 */
export const GetMediaServerParams = zod.object({
  id: zod.uuid(),
});

export const GetMediaServerResponse = MediaserversServerView;

/**
 * @summary Change a Media Server, testing it again when the change is to its address, reaching Node or API key
 */
export const UpdateMediaServerParams = zod.object({
  id: zod.uuid(),
});

export const UpdateMediaServerBody = MediaserversPatchBody;

export const UpdateMediaServerResponse = MediaserversServerView;

/**
 * @summary List a Media Server's recent changes, newest first, with their results and what each move couldn't carry
 */
export const ListMediaServerChangesParams = zod.object({
  id: zod.uuid(),
});

export const listMediaServerChangesQueryCursorMax = 256;

export const ListMediaServerChangesQueryParams = zod.object({
  cursor: zod
    .string()
    .max(listMediaServerChangesQueryCursorMax)
    .exactOptional()
    .describe("The next member of the previous page; absent for the newest"),
});

export const ListMediaServerChangesResponse = MediaserversChanges;

/**
 * @summary Accept the server that responds at a Media Server's address now as that Media Server
 */
export const ConfirmMediaServerIdentityParams = zod.object({
  id: zod.uuid(),
});

export const ConfirmMediaServerIdentityResponse = MediaserversServerView;

/**
 * @summary List every Node that fetched a Media Server's credential
 */
export const ListMediaServerCredentialHoldersParams = zod.object({
  id: zod.uuid(),
});

export const ListMediaServerCredentialHoldersResponse = HoldersOutputBody;

/**
 * @summary List a Media Server's library folders and every library root's Path mapping, pre-filled where none is saved
 */
export const ListMediaServerLibrariesParams = zod.object({
  id: zod.uuid(),
});

export const ListMediaServerLibrariesResponse = MediaserversLibraries;

/**
 * @summary Save a Media Server's Path mappings once each shown folder sits inside one of its library folders
 */
export const SaveMediaServerMappingsParams = zod.object({
  id: zod.uuid(),
});

export const SaveMediaServerMappingsBody = MediaserversMappingsBody;

export const SaveMediaServerMappingsResponse = MappingsOutputBody;

/**
 * @summary Queue a whole library refresh of every library a Media Server's confirmed Path mappings lead into
 */
export const RefreshMediaServerParams = zod.object({
  id: zod.uuid(),
});

export const RefreshMediaServerResponse = zod.void();

/**
 * @summary List what plays on a Media Server now, as its reaching Node last saw it
 */
export const ListMediaServerSessionsParams = zod.object({
  id: zod.uuid(),
});

export const ListMediaServerSessionsResponse = MediaserversSessions;

/**
 * @summary Test a Media Server from its reaching Node at its address and from the Controller at its fallback address
 */
export const TestMediaServerParams = zod.object({
  id: zod.uuid(),
});

export const TestMediaServerResponse = MediaserversTested;
