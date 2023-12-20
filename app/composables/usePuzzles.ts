import { defineStore } from 'pinia';

export default defineStore("puzzles", {
  state: () => ({
    _rounds: {} as { [id: number]: Round; },
    _puzzles: {} as { [id: number]: Puzzle; },
    next_hunt: undefined as Date | undefined,
  }),
  getters: {
    rounds(state): AnnotatedRound[] {
      const rounds: AnnotatedRound[] = [];
      for (const base of Object.values(state._rounds)) {
        const puzzles = this.puzzles.get(base.id) || [];
        rounds.push({
          ...base,
          anchor: base.name.trim().toLowerCase().replaceAll(" ", "-"),
          complete: puzzles.filter((p => !p.answer)).length === 0,
          displayName: `${base.emoji}\uFE0F ${base.name}`,
          solved: puzzles.filter((p) => !!p.answer).length,
          total: puzzles.length,
        });
      }
      rounds.sort((a, b) => {
        if (a.special && !b.special) return 1;
        else if (!a.special && b.special) return -1;
        else return a.id - b.id;
      });
      return rounds;
    },
    puzzles(state): Map<number, Puzzle[]> {
      const grouped = new Map<number, Puzzle[]>();
      for (const puzzle of Object.values(state._puzzles)) {
        const id = puzzle.round.id;
        if (!grouped.has(id)) {
          grouped.set(id, []);
        }
        grouped.get(id)!.push(puzzle);
      }
      for (const [_, puzzles] of grouped) {
        puzzles.sort((a, b) => {
          if (a.meta && !b.meta) return 1;
          else if (!a.meta && b.meta) return -1;
          else return a.name.localeCompare(b.name);
        });
      }
      return grouped;
    },
  },
  actions: {
    async refresh() {
      const data = await useAPI<any>("/home");
      this._rounds = {};
      for (const round of data.value.rounds || []) {
        this._rounds[round.id] = round;
      }
      this._puzzles = {};
      for (const puzzle of data.value.puzzles || []) {
        this._puzzles[puzzle.id] = puzzle;
      }
      this.next_hunt = data.value.next_hunt ?
        new Date(data.value.next_hunt) : undefined;
    },
    async addRound(data: Omit<Round, "id">) {
      return useAPI(`/rounds`, data)
        .then((r: any) => r.value && (this._rounds[r.value.id] = r.value));
    },
    async updateRound(round: Round, data: Partial<Round>) {
      const previous = this._rounds[round.id];
      this._rounds[round.id] = { ...previous, ...data };
      await useAPI(`/rounds/${round.id}`, data)
        .catch(() => this._rounds[round.id] = previous);
    },
    async addPuzzle(data: { name: string; round: number; puzzle_url: string; }) {
      return useAPI(`/puzzles`, data)
        .then((r: any) => r.value && (this._puzzles[r.value.id] = r.value));
    },
    async updatePuzzle(puzzle: Puzzle, data: Partial<Puzzle>) {
      const previous = this._puzzles[puzzle.id];
      this._puzzles[puzzle.id] = { ...previous, ...data };
      await useAPI(`/puzzles/${puzzle.id}`, data)
        .catch(() => this._puzzles[puzzle.id] = previous);
    },
  },
});
