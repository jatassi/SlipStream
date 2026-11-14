import type {
  MediaserversSessionView as Session,
  MediaserversSessionViewDecision as Decision,
} from "../../api/gen/model";

// Jellyfin and Emby name their reasons from one TranscodeReason enum. A name
// this table lacks still shows as the server gave it, under Details.
const reasonWords: Readonly<Record<string, string>> = {
  ContainerNotSupported: "the player can't play its container",
  VideoCodecNotSupported: "the player can't play its video codec",
  AudioCodecNotSupported: "the player can't play its audio codec",
  SubtitleCodecNotSupported: "the player can't show its subtitles",
  AudioIsExternal: "its audio is in a separate file",
  SecondaryAudioNotSupported: "the player can't play a second audio track",
  VideoProfileNotSupported: "the player can't play its video profile",
  VideoLevelNotSupported: "the player can't play its video level",
  VideoResolutionNotSupported: "the player can't play its resolution",
  VideoBitDepthNotSupported: "the player can't play its video bit depth",
  VideoFramerateNotSupported: "the player can't play its frame rate",
  VideoRangeTypeNotSupported: "the player can't show its HDR format",
  VideoCodecTagNotSupported: "the player can't play its video codec tag",
  RefFramesNotSupported: "the player can't play its reference frames",
  AnamorphicVideoNotSupported: "the player can't play anamorphic video",
  InterlacedVideoNotSupported: "the player can't play interlaced video",
  AudioChannelsNotSupported: "the player can't play its audio channels",
  AudioProfileNotSupported: "the player can't play its audio profile",
  AudioSampleRateNotSupported: "the player can't play its audio sample rate",
  AudioBitDepthNotSupported: "the player can't play its audio bit depth",
  ContainerBitrateExceedsLimit: "its bitrate is above the player's limit",
  VideoBitrateNotSupported: "its video bitrate is above the player's limit",
  AudioBitrateNotSupported: "its audio bitrate is above the player's limit",
  StreamCountExceedsLimit: "it has more streams than the player allows",
  UnknownVideoStreamInfo: "the server can't read its video stream",
  UnknownAudioStreamInfo: "the server can't read its audio stream",
  DirectPlayError: "direct play failed on the player",
};

export const decisionLabels: { readonly [D in Decision]: string } = {
  direct_play: "Direct play",
  copy: "Copy",
  transcode: "Transcode",
};

const list = new Intl.ListFormat("en", { type: "conjunction" });

/** What the server does with the file, and why when it converts it. */
export function decisionWords(server: string, session: Session): string {
  if (session.decision === "direct_play") {
    return `${server} sends the file as it is.`;
  }
  if (session.decision === "copy") {
    return `${server} copies the file's streams into a new container without converting them.`;
  }
  const where = session.hardware ? " on its graphics hardware" : "";
  const known = session.reasons.flatMap((reason) => reasonWords[reason] ?? []);
  const why = known.length === 0 ? "" : ` because ${list.format(known)}`;
  return `${server} converts the file${where}${why}.`;
}
