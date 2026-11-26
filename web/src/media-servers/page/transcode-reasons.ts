import {
  ArrowReloadHorizontalIcon,
  CpuIcon,
  PackageMovingIcon,
  PlayIcon,
} from "@hugeicons/core-free-icons";

import type {
  MediaserversSessionView as Session,
  MediaserversSessionViewDecision as Decision,
} from "../../api/gen/model";
import type { StatusMark, TermsFor } from "../../ui/state-marks";

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

type Shown = Decision | "hardware_transcode";

const decisionTerms = {
  direct_play: {
    word: "Direct play",
    icon: PlayIcon,
    tone: "neutral",
    definition: "The player plays the original file.",
  },
  direct_stream: {
    word: "Direct stream",
    icon: PackageMovingIcon,
    tone: "neutral",
    definition: "The server repackages the audio and video for the player without converting them.",
  },
  transcode: {
    word: "Transcode",
    icon: ArrowReloadHorizontalIcon,
    tone: "neutral",
    definition: "The server converts the file into a format the player can play.",
  },
  hardware_transcode: {
    word: "Hardware transcode",
    icon: CpuIcon,
    tone: "neutral",
    definition:
      "The server converts the file on its graphics hardware into a format the player can play.",
  },
} as const satisfies TermsFor<Shown, "neutral">;

/** The standard name for what the server does with the file, with its definition. */
export function decisionTerm(session: Session): StatusMark {
  return decisionTerms[
    session.decision === "transcode" && session.hardware ? "hardware_transcode" : session.decision
  ];
}

const list = new Intl.ListFormat("en", { type: "conjunction" });

/** Why the server transcodes, in plain words, or undefined when it gave no known reason. */
export function transcodeWhy(session: Session): string | undefined {
  if (session.decision !== "transcode") {
    return undefined;
  }
  const known = session.reasons.flatMap((reason) => reasonWords[reason] ?? []);
  return known.length === 0 ? undefined : `Transcoding because ${list.format(known)}.`;
}
