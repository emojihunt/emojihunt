export type FocusInfo = { index: number; };

export type Round = {
  id: number;
  name: string;
  emoji: string;
};

export type Puzzle = {
  id: number;
  name: string;
  answer: string;
  round: Round;
  status: Status;
  description: string;
  location: string;
  puzzle_url: string;
  spreadsheet_id: string;
  discord_channel: string;
  original_url: string;
  name_override: string;
  archived: boolean;
  voice_room: string;
};

export type RoundStats = Round & {
  anchor: string;
  complete: boolean;
  hue: number;
  solved: number;
  total: number;
};

export enum Status {
  NotStarted = "",
  Working = "Working",
  Abandoned = "Abandoned",
  Solved = "Solved",
  Backsolved = "Backsolved",
  Purchased = "Purchased",
}

export const StatusLabel = (status: Status): string => status || "Not Started";

export const StatusEmoji = (status: Status): string => {
  switch (status) {
    case Status.NotStarted: return "";
    case Status.Working: return "✍️";
    case Status.Abandoned: return "🗑️";
    case Status.Solved: return "🏅";
    case Status.Backsolved: return "🤦‍♀️";
    case Status.Purchased: return "💸";
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
