import { defineStore } from 'pinia';

export default defineStore("puzzles", {
  state: () => ({
    _rounds: new Map<number, Round>(),
    _puzzles: new Map<number, Puzzle>(),
    next_hunt: undefined as Date | undefined,
    voice_rooms: {} as Record<string, string>
  }),
  getters: {
    rounds(state): AnnotatedRound[] {
      const rounds: AnnotatedRound[] = [];
      for (const base of state._rounds.values()) {
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
        if (a.special !== b.special) return a.special ? 1 : -1;
        else if (a.sort !== b.sort) return a.sort - b.sort;
        else return a.id - b.id;
      });
      return rounds;
    },
    puzzles(state): Map<number, Puzzle[]> {
      const grouped = new Map<number, Puzzle[]>();
      for (const puzzle of state._puzzles.values()) {
        const id = puzzle.round.id;
        if (!grouped.has(id)) {
          grouped.set(id, []);
        }
        grouped.get(id)!.push(puzzle);
      }
      for (const [_, puzzles] of grouped) {
        puzzles.sort((a, b) => {
          if (a.meta !== b.meta) return a.meta ? 1 : -1;
          else return a.name.localeCompare(b.name);
        });
      }
      return grouped;
    },
  },
  actions: {
    async refresh() {
      const data = await useAPI<any>("/home");
      this._rounds.clear();
      this._puzzles.clear();
      (data.value.rounds || []).forEach((r: any) => this._rounds.set(r.id, r));
      (data.value.puzzles || []).forEach((p: any) => this._puzzles.set(p.id, p));
      this.next_hunt = data.value.next_hunt ?
        new Date(data.value.next_hunt) : undefined;
      this.voice_rooms = data.value.voice_rooms;
    },
    async addRound(data: { name: string; emoji: string; hue: number; }) {
      return useAPI(`/rounds`, data)
        .then((r: any) => r.value && this._rounds.set(r.value.id, r.value));
    },
    async updateRound(round: Round, data: Omit<Partial<Round>, "id">) {
      const previous = this._rounds.get(round.id)!;
      this._rounds.set(round.id, { ...previous, ...data });
      await useAPI(`/rounds/${round.id}`, data)
        .catch(() => this._rounds.set(round.id, previous));
    },
    async addPuzzle(data: { name: string; round: number; puzzle_url: string; }) {
      return useAPI(`/puzzles`, data)
        .then((r: any) => r.value && this._puzzles.set(r.value.id, r.value));
    },
    async updatePuzzle(puzzle: Puzzle, data: Omit<Partial<Puzzle>, "id">) {
      const previous = this._puzzles.get(puzzle.id)!;
      this._puzzles.set(puzzle.id, { ...previous, ...data });
      await useAPI(`/puzzles/${puzzle.id}`, data)
        .catch(() => this._puzzles.set(puzzle.id, previous));
    },
  },
});
