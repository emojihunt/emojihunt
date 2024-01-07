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

export type ServerPuzzle = Omit<Puzzle, "round"> & { round: Round; };

export type HomeResponse = {
  puzzles: ServerPuzzle[];
  rounds: Round[];
  change_id: number;
  next_hunt: string | undefined;
  voice_rooms: Record<string, string>;
};

export type AblyWorkerMessage =
  { name: "sync"; data: SyncMessage; } |
  { name: "client"; state: ConnectionState; };

export type ConnectionState = "disconnected" | "connected" | "broken";

export type SyncMessage = {
  change_id: number;
  kind: "upsert" | "delete";
  puzzle?: ServerPuzzle;
  round?: Round;
  reminder_fix?: string;
};

export type FocusInfo = { index: number; };
