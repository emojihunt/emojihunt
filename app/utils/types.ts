type FocusInfo = { index: number; };

type Round = {
  id: number;
  name: string;
  emoji: string;
};

type Puzzle = {
  id: number;
  name: string;
  answer: string;
  round: Round;
  status: string;
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

type RoundStats = Round & {
  anchor: string;
  complete: boolean;
  hue: number;
  solved: number;
  total: number;
};
