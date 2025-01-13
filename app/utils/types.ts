export type Round = {
  id: number;
  name: string;
  emoji: string;
  hue: number;
  sort: number;
  special: boolean;
  drive_folder: string;
  discord_category: string;
};

export const RoundKeys: (keyof Omit<Round, "id">)[] = [
  "name", "emoji", "hue", "sort", "special", "drive_folder", "discord_category",
];

export type AnnotatedRound = Round & {
  anchor: string;
  complete: boolean;
  displayName: string;
  solved: number;
  total: number;
};

export type NewRound = {
  name: string;
  emoji: string;
  hue: number;
  sort?: number;
  special?: boolean;
  drive_folder?: string;
  discord_category?: string;
};

export type Puzzle = {
  id: number;
  name: string;
  answer: string;
  round: number;
  status: Status;
  note: string;
  location: string;
  puzzle_url: string;
  spreadsheet_id: string;
  discord_channel: string;
  meta: boolean;
  voice_room: string;
  reminder: string;
};

export const PuzzleKeys: (keyof Omit<Puzzle, "id">)[] = [
  "name", "answer", "round", "status", "note", "location", "puzzle_url",
  "spreadsheet_id", "discord_channel", "meta", "voice_room", "reminder",
];

export type NewPuzzle = {
  name: string;
  round: number;
  puzzle_url: string;
  spreadsheet_id?: string;
  discord_channel?: string;
  meta?: boolean;
};

export enum Status {
  NotStarted = "",
  Working = "Working",
  Abandoned = "Abandoned",
  Solved = "Solved",
  Backsolved = "Backsolved",
  Purchased = "Purchased",
};

// https://stackoverflow.com/a/62850363
export const Statuses: Status[] = Object.keys(Status).
  filter((k) => !isFinite(Number(k))).
  map((k) => (Status as any)[k]);

export const StatusLabel = (status: Status): string => status || "Not Started";

export const StatusEmoji = (status: Status): string => {
  switch (status) {
    case Status.NotStarted: return "";
    case Status.Working: return "✍️";
    case Status.Abandoned: return "🗑️";
    case Status.Solved: return "🏅";
    case Status.Backsolved: return "🤦‍♀️";
    case Status.Purchased: return "💸 ";
  }
};

export const StatusNeedsAnswer = (status: Status): boolean => {
  switch (status) {
    case Status.NotStarted: return false;
    case Status.Working: return false;
    case Status.Abandoned: return false;
    case Status.Solved: return true;
    case Status.Backsolved: return true;
    case Status.Purchased: return true;
  }
};

export const DefaultReminder = "0001-01-01T00:00:00Z";

export const parseTimestamp = (timestamp: string): Date | null => {
  if (!timestamp) return null;
  const date = new Date(timestamp);
  if (date.getTime() < 1700000000000) return null;
  return date;
};

export type HomeResponse = {
  change_id: number;
  puzzles: Puzzle[];
  rounds: Round[];
  settings: SettingsMessage;
};

export type SettingsMessage = {
  hunt_name: string;
  hunt_url: string;
  hunt_credentials: string;
  logistics_url: string;
  discord_guild: string;
  hanging_out: string;
  next_hunt: string;
  voice_rooms: Record<string, string>;
};

export type DiscoveryConfig = {
  puzzles_url: string;
  cookie_name: string;
  cookie_value: string;
  group_mode: boolean;
  group_selector: string;
  round_name_selector: string;
  puzzle_list_selector: string;
  puzzle_item_selector: string;
  websocket_url: string;
  websocket_token: string;
  hunt_name: string;
  hunt_url: string;
  hunt_credentials: string;
  logistics_url: string;
};

export type ScrapedPuzzle = {
  name: string;
  round_name: string;
  puzzle_url: string;
};

export type VoiceRoom = {
  id: string;
  emoji: string;
  name: string;
};

export type AblyWorkerMessage =
  { name: "sync"; data: SyncMessage; } |
  { name: "settings"; data: SettingsMessage; } |
  { name: "m"; data: DiscordMessage; } |
  { name: "client"; state: ConnectionState; };

export type ConnectionState = "disconnected" | "connected" | "broken";

export type SyncMessage = {
  change_id: number;
  kind: "upsert" | "delete";
  puzzle?: Puzzle;
  round?: Round;
};

export type DiscordMessage = {
  id: string;
  ch: string;
  u: {
    id: string;
    name: string;
    avatar: string;
  };
  msg: string;
};

export const ObserverKey = Symbol() as
  InjectionKey<Ref<IntersectionObserver | undefined>>;

export const ExpandedKey = Symbol() as InjectionKey<Ref<number>>;
