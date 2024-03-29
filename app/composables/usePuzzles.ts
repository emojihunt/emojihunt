import { defineStore } from 'pinia';
import type { HomeResponse } from '~/utils/types';

type Optimistic = (
  ({ type: "round"; } & Partial<Round>) |
  ({ type: "round.delete", id: number; }) |
  ({ type: "puzzle"; } & Partial<Puzzle>) |
  ({ type: "puzzle.delete", id: number; })
) & { id: number; };

const updateRequest = async <T>(endpoint: string, params: any): Promise<[T, number]> => {
  let args: RequestInit;
  if (params.delete === true) {
    args = {
      method: "DELETE",
      // Workaround for https://github.com/nuxt/nuxt/issues/23422
      headers: {
        "Content-Type": "application/x-www-form-urlencoded",
      },
      body: "-",
    };
  } else {
    args = {
      method: "POST",
      headers: {
        "Content-Type": "application/x-www-form-urlencoded",
      },
      body: (new URLSearchParams(params as any)).toString(),
    };
  }
  const response = await fetch(`/api${endpoint}`, args);
  if (response.status === 401) {
    window.location.reload();
  } else if (response.status !== 200) {
    throw createError({
      fatal: true,
      statusCode: response.status,
      data: await response.json().catch(() => response.text()),
    });
  }
  const header = response.headers.get("X-Change-ID");
  if (!header) throw "Missing X-Change-ID header";
  return [await response.json(), parseInt(header)];
};

export default defineStore("puzzles", {
  state: () => ({
    _rounds: new Map<number, Round>(),
    _puzzles: new Map<number, Puzzle>(),
    _initialChangeId: 0,
    nextHunt: undefined as Date | undefined,
    voiceRooms: {} as Record<string, string>,

    // Writes not yet received from Ably. Committed writes first (earlier, by
    // change ID); pending writes second (later, by local ID).
    _optimisticCounter: Math.floor(Number.MAX_SAFE_INTEGER / 2),
    _optimistic: new Map<number, Optimistic>(),
  }),
  getters: {
    rounds(state): AnnotatedRound[] {
      const rounds = new Map(state._rounds);
      const entries = [...state._optimistic.entries()].sort();
      for (const [_, entry] of entries) {
        if (entry.type === "round") {
          rounds.set(entry.id, { ...rounds.get(entry.id)!, ...entry });
        } else if (entry.type === "round.delete") {
          rounds.delete(entry.id);
        }
      }
      const annotated: AnnotatedRound[] = [];
      for (const base of rounds.values()) {
        const puzzles = this.puzzlesByRound.get(base.id) || [];
        annotated.push({
          ...base,
          anchor: base.name.trim().toLowerCase().replaceAll(/[^A-Za-z0-9]+/g, "-"),
          complete: puzzles.filter((p => !p.answer)).length === 0,
          displayName: `${base.emoji}\uFE0F ${base.name}`,
          solved: puzzles.filter((p) => !!p.answer).length,
          total: puzzles.length,
        });
      }
      annotated.sort((a, b) => {
        if (a.special !== b.special) return a.special ? -1 : 1;
        else if (a.sort !== b.sort) return a.sort - b.sort;
        else return a.id - b.id;
      });
      return annotated;
    },
    puzzles(state): Map<number, Puzzle> {
      const puzzles = new Map(state._puzzles);
      const entries = [...state._optimistic.entries()].sort();
      for (const [_, entry] of entries) {
        if (entry.type === "puzzle") {
          puzzles.set(entry.id, { ...puzzles.get(entry.id)!, ...entry });
        } else if (entry.type === "puzzle.delete") {
          puzzles.delete(entry.id);
        }
      }
      return puzzles;
    },
    puzzlesByRound(): Map<number, Puzzle[]> {
      const grouped = new Map<number, Puzzle[]>();
      for (const puzzle of this.puzzles.values()) {
        if (!grouped.has(puzzle.round)) {
          grouped.set(puzzle.round, []);
        }
        grouped.get(puzzle.round)!.push(puzzle);
      }
      for (const [_, puzzles] of grouped) {
        puzzles.sort((a, b) => {
          if (a.meta !== b.meta) return a.meta ? 1 : -1;
          else return a.name.localeCompare(b.name);
        });
      }
      return grouped;
    },
    puzzleCount(): number {
      let count = 0;
      for (const [_round, puzzles] of this.puzzlesByRound) {
        count += puzzles.length;
      }
      return count;
    },
  },
  actions: {
    async refresh() {
      if (import.meta.server && !useCookie("session").value) {
        throw createError({
          message: "short-circuiting to login page",
          statusCode: 401,
        });
      }
      const { data, error } = await useFetch<HomeResponse>("/api/home");
      if (error.value) {
        throw createError({
          fatal: true,
          message: error.value.message,
          statusCode: error.value.statusCode,
          data: error.value.data,
        });
      }
      this._rounds.clear();
      this._puzzles.clear();
      this._optimistic.clear();
      (data.value?.rounds || []).forEach((r: any) => this._rounds.set(r.id, r));
      (data.value?.puzzles || []).forEach((p: any) => this._puzzles.set(p.id, { ...p, round: p.round.id }));
      this._initialChangeId = data.value?.change_id || 0;
      this.nextHunt = data.value?.next_hunt ?
        new Date(data.value.next_hunt) : undefined;
      this.voiceRooms = data.value?.voice_rooms || {};
    },
    async addRound(data: NewRound) {
      const [round, changeId] = await updateRequest<Round>("/rounds", data);
      this._optimistic.set(changeId, { type: "round", ...round });
    },
    async updateRound(id: number, data: Omit<Partial<Round>, "id">) {
      const [_, changeId] = await updateRequest<Round>(`/rounds/${id}`, data);
      this._optimistic.set(changeId, { type: "round", id, ...data });
    },
    async updateRoundOptimistic(id: number, data: Omit<Partial<Round>, "id">) {
      const localId = this._optimisticCounter++;
      const delta: Optimistic = { type: "round", id, ...data };
      this._optimistic.set(localId, delta);
      try {
        const [_, changeId] = await updateRequest<Round>(`/rounds/${id}`, data);
        this._optimistic.set(changeId, delta);
      } finally { this._optimistic.delete(localId); }
    },
    async deleteRound(id: number) {
      const [_, changeId] = await updateRequest<Puzzle>(
        `/rounds/${id}`, { delete: true });
      this._optimistic.set(changeId, { type: "round.delete", id });
    },
    async addPuzzle(data: NewPuzzle) {
      const [puzzle, changeId] = await updateRequest<Puzzle>("/puzzles", data);
      this._optimistic.set(changeId, { type: "puzzle", ...puzzle });
    },
    async updatePuzzle(id: number, data: Omit<Partial<Puzzle>, "id">) {
      const [_, changeId] = await updateRequest<Puzzle>(`/puzzles/${id}`, data);
      this._optimistic.set(changeId, { type: "puzzle", id, ...data });
    },
    async updatePuzzleOptimistic(id: number, data: Omit<Partial<Puzzle>, "id">) {
      const localId = this._optimisticCounter++;
      const delta: Optimistic = { type: "puzzle", id, ...data };
      this._optimistic.set(localId, delta);
      try {
        const [_, changeId] = await updateRequest<Puzzle>(`/puzzles/${id}`, data);
        this._optimistic.set(changeId, delta);
      } finally { this._optimistic.delete(localId); }
    },
    async deletePuzzle(id: number) {
      const [_, changeId] = await updateRequest<Puzzle>(
        `/puzzles/${id}`, { delete: true });
      this._optimistic.set(changeId, { type: "puzzle.delete", id });
    },
    handleDelta({ change_id, kind, puzzle, round, reminder_fix }: SyncMessage) {
      this._optimistic.delete(change_id);
      if (change_id <= this._initialChangeId) return;
      if (kind === "upsert") {
        if (puzzle) {
          puzzle.reminder = reminder_fix!;
          this._puzzles.set(puzzle.id, { ...puzzle, round: puzzle.round.id });
        }
        if (round) this._rounds.set(round.id, round);
      } else if (kind === "delete") {
        if (puzzle) this._puzzles.delete(puzzle.id);
        if (round) this._rounds.delete(round.id);
      } else {
        console.error(`unknown update kind: ${kind}`);
      }
    },
  },
});
