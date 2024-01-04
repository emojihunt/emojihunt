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
  round: Round;
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

export type HomeResponse = {
  puzzles: Puzzle[];
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
  puzzle?: Puzzle;
  round?: Round;
  reminder_fix?: string;
};

export type FocusInfo = { index: number; };
