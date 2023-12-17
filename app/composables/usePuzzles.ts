import { defineStore } from 'pinia';

// HACK: apply hard-coded colors to rounds for testing
const hues: { [round: string]: number; } = {
  "1": 241, "2": 178, "3": 80, "4": 45,
  "5": 255, "6": 19, "7": 69, "8": 205,
  "9": 28, "10": 24, "11": 141,
};

export default defineStore("puzzles", {
  state: () => ({ puzzles: {} as { [id: number]: Puzzle; } }),
  getters: {
    puzzlesByRound(state): { [round: string]: Puzzle[]; } {
      const grouped: { [round: string]: Puzzle[]; } = {};
      for (const puzzle of Object.values(state.puzzles)) {
        const id = puzzle.round.id;
        grouped[id] ||= [];
        grouped[id].push(puzzle);
      }
      return grouped;
    },
    roundStats(): { [round: string]: RoundStats; } {
      const stats: { [round: string]: RoundStats; } = {};
      for (const id of Object.keys(this.puzzlesByRound)) {
        const example = this.puzzlesByRound[id][0].round;
        stats[id] = {
          anchor: example.name.trim().toLowerCase().replaceAll(" ", "-"),
          complete: this.puzzlesByRound[id].filter((p => !p.answer)).length == 0,
          hue: hues[id],
          solved: this.puzzlesByRound[id].filter((p) => !!p.answer).length,
          total: this.puzzlesByRound[id].length,

          id: example.id,
          name: example.name.trim(),
          emoji: example.emoji,
        };
      }
      return stats;
    },
  },
  actions: {
    async refresh() {
      const data = await useAPI<Puzzle[]>("/puzzles");
      for (const puzzle of data.value) {
        this.puzzles[puzzle.id] = puzzle;
      }
    },
    async updatePuzzle(puzzle: Puzzle, data: Partial<Puzzle>) {
      await useAPI(`/puzzles/${puzzle.id}`, data);
      this.puzzles[puzzle.id] = { ...data, ...this.puzzles[puzzle.id] };
    },
  },
});
