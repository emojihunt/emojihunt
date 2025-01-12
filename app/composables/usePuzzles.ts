// Reactive puzzle state (public interface).
//
// It's safe to destructure the immediate keys of this object.
//
type State = {
  connected: Ref<boolean>,

  settings: Settings;
  puzzles: Map<number, Puzzle>;
  rounds: Map<number, AnnotatedRound>;
  voiceRooms: Map<string, VoiceRoom>;

  ordering: Ref<SortedRound[]>;
  puzzleCount: Ref<number>;
  solvedPuzzleCount: Ref<number>;

  addRound: (data: NewRound) => Promise<void>;
  updateRound: (id: number, data: Omit<Partial<Round>, "id">) => Promise<void>;
  updateRoundOptimistic: (id: number, data: Omit<Partial<Round>, "id">) => Promise<void>;
  deleteRound: (id: number) => Promise<void>;
  addPuzzle: (data: NewPuzzle) => Promise<void>;
  updatePuzzle: (id: number, data: Omit<Partial<Puzzle>, "id">) => Promise<void>;
  updatePuzzleOptimistic: (id: number, data: Omit<Partial<Puzzle>, "id">) => Promise<void>;
  deletePuzzle: (id: number) => Promise<void>;
};

type SortedRound = AnnotatedRound & { puzzles: Puzzle[]; };

type Settings = {
  discordGuild: string;
  hangingOut: string;
  huntName: string;
  huntURL: string;
  huntCredentials: string;
  logisticsURL: string;
  nextHunt: Date | null;
};

type Optimistic = (
  ({ type: "puzzle"; } & Partial<Puzzle>) |
  ({ type: "round"; } & Partial<Round>) |
  ({ type: "puzzle.delete", id: number; }) |
  ({ type: "round.delete", id: number; })
) & { id: number; };

const key = Symbol() as InjectionKey<State>;

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

const updateReactiveMap = <K, V>(m: Map<K, V>, k: K, v: V): void => {
  const existing = m.get(k);
  if (existing) Object.assign(existing, v);
  else m.set(k, v);
};

const hydrateRound = (raw: Round, puzzles: Puzzle[]): AnnotatedRound => {
  const metas = puzzles.filter((p => p.meta));
  return {
    ...raw,
    anchor: raw.name.trim().toLowerCase().replaceAll(/[^A-Za-z0-9]+/g, "-"),
    complete: puzzles.length > 0 &&
      (metas.length === 0 ? puzzles : metas).filter((p => !p.answer)).length === 0,
    displayName: `${raw.emoji}\uFE0F ${raw.name}`,
    solved: puzzles.filter((p) => p.answer).length,
    total: puzzles.length,
  };
};

// Called in the top-level component.
//
// Note: using `provide` and `inject` in the same component doesn't work! So
// `initializePuzzles` may only be called in the top-level component, and
// `usePuzzles` may only be called in lower-level components.
//
// https://github.com/vuejs/vue/issues/12678
//
export async function initializePuzzles(): Promise<State> {
  if (inject(key, undefined)) {
    throw new Error("usePuzzles() may only be initialized once");
  }
  if (import.meta.server && !useCookie("session").value) {
    throw createError({
      message: "short-circuiting to login page",
      statusCode: 401,
    });
  }

  const settings: Settings = reactive({
    discordGuild: "", hangingOut: "", huntName: "", huntURL: "",
    huntCredentials: "", logisticsURL: "", nextHunt: null,
  });
  const _puzzles = new Map<number, Puzzle>();
  const _rounds = new Map<number, Round>();
  const puzzles = reactive(new Map<number, Puzzle>());
  const rounds = reactive(new Map<number, AnnotatedRound>());
  const voiceRooms = reactive(new Map<string, VoiceRoom>());
  const ordering = ref<SortedRound[]>([]);
  const puzzleCount = ref(0);
  const solvedPuzzleCount = ref(0);

  const optimistic = new Map<number, Optimistic>();
  let optimisticCounter = Math.floor(Number.MAX_SAFE_INTEGER / 2);
  let initialChangeId = 0;

  const refresh = () => { // clocks at around 4ms
    // First, materialize optimistically-applied updates.
    const localPuzzles = new Map(_puzzles);
    const localRounds = new Map(_rounds);
    for (const [_, entry] of [...optimistic.entries()].sort()) {
      switch (entry.type) {
        case "puzzle":
          localPuzzles.set(entry.id, { ..._puzzles.get(entry.id)!, ...entry });
          break;
        case "round":
          localRounds.set(entry.id, { ...rounds.get(entry.id)!, ...entry });
          break;
        case "puzzle.delete":
          localPuzzles.delete(entry.id);
          break;
        case "round.delete":
          localRounds.delete(entry.id);
          break;
      }
    }

    localPuzzles.forEach((v, k) => updateReactiveMap(puzzles, k, v));
    puzzles.forEach((_, k) => _puzzles.has(k) || puzzles.delete(k));

    const grouped = new Map<number, Puzzle[]>();
    for (const puzzle of puzzles.values()) {
      const g = grouped.get(puzzle.round);
      if (g) g.push(puzzle);
      else grouped.set(puzzle.round, [puzzle]);
    }
    for (const [_, puzzles] of grouped) {
      puzzles.sort((a, b) => {
        if (a.meta !== b.meta) return a.meta ? 1 : -1;
        const ra = parseReminder(a);
        const rb = parseReminder(b);
        if (ra) {
          if (rb) return ra.getTime() - rb.getTime();
          if (rb) return a.name.localeCompare(b.name);
          else return -1;
        } else {
          if (rb) return 1;
          else return a.name.localeCompare(b.name);
        }
      });
    }

    localRounds.forEach((v, k) => updateReactiveMap(rounds, k, hydrateRound(v, grouped.get(v.id) || [])));
    rounds.forEach((_, k) => _rounds.has(k) || rounds.delete(k));

    puzzleCount.value = puzzles.size;

    solvedPuzzleCount.value = 0;
    puzzles.forEach((p) => p.answer && (solvedPuzzleCount.value++));

    ordering.value = [...rounds.values()].map((r) => ({ ...r, puzzles: grouped.get(r.id) || [] }));
    ordering.value.sort((a, b) => {
      if (a.special !== b.special) return a.special ? -1 : 1;
      else if (a.sort !== b.sort) return a.sort - b.sort;
      else return a.id - b.id;
    });
  };

  const onSync = ({ change_id, kind, puzzle, round, reminder_fix }: SyncMessage) => {
    optimistic.delete(change_id);
    if (change_id <= initialChangeId) return;
    if (kind === "upsert") {
      if (puzzle) {
        puzzle.reminder = reminder_fix!;
        _puzzles.set(puzzle.id, puzzle);
      }
      if (round) _rounds.set(round.id, round);
    } else if (kind === "delete") {
      if (puzzle) _puzzles.delete(puzzle.id);
      if (round) _rounds.delete(round.id);
    } else {
      console.error(`unknown update kind: ${kind}`);
    }
    refresh();
  };
  const onSettings = (msg: SettingsMessage) => {
    settings.huntName = msg.hunt_name;
    settings.huntURL = msg.hunt_url;
    settings.huntCredentials = msg.hunt_credentials;
    settings.logisticsURL = msg.logistics_url;
    settings.discordGuild = msg.discord_guild;
    settings.hangingOut = msg.hanging_out;
    settings.nextHunt = msg.next_hunt ? new Date(msg.next_hunt) : null;

    Object.entries(msg.voice_rooms).forEach(([id, raw]) => {
      // We expect the channel's emoji to go at the end
      const p = raw.split(" ");
      if ([...p[p.length - 1]].length === 1) {
        const name = p.slice(0, p.length - 1).join(" ");
        const emoji = p[p.length - 1];
        voiceRooms.set(id, { id, name, emoji });
      } else {
        voiceRooms.set(id, { id, name: raw, emoji: "📻" });
      }
    });
    const ids = new Set<string>(Object.keys(msg.voice_rooms));
    voiceRooms.forEach((_, i) => !ids.has(i) && voiceRooms.delete(i));
  };
  const connected = useAbly(onSync, onSettings);

  const state = {
    connected, settings, puzzles, rounds, voiceRooms,
    puzzleCount, solvedPuzzleCount, ordering,
    async addRound(data: NewRound) {
      const [round, changeId] = await updateRequest<Round>("/rounds", data);
      optimistic.set(changeId, { type: "round", ...round });
      refresh();
    },
    async updateRound(id: number, data: Omit<Partial<Round>, "id">) {
      const [_, changeId] = await updateRequest<Round>(`/rounds/${id}`, data);
      optimistic.set(changeId, { type: "round", id, ...data });
      refresh();
    },
    async updateRoundOptimistic(id: number, data: Omit<Partial<Round>, "id">) {
      const localId = optimisticCounter++;
      const delta: Optimistic = { type: "round", id, ...data };
      optimistic.set(localId, delta);
      refresh();
      try {
        const [_, changeId] = await updateRequest<Round>(`/rounds/${id}`, data);
        optimistic.set(changeId, delta);
      } finally {
        optimistic.delete(localId);
        refresh();
      }
    },
    async deleteRound(id: number) {
      const [_, changeId] = await updateRequest<Puzzle>(
        `/rounds/${id}`, { delete: true });
      optimistic.set(changeId, { type: "round.delete", id });
      refresh();
    },
    async addPuzzle(data: NewPuzzle) {
      const [puzzle, changeId] = await updateRequest<Puzzle>("/puzzles", data);
      optimistic.set(changeId, { type: "puzzle", ...puzzle });
      refresh();
    },
    async updatePuzzle(id: number, data: Omit<Partial<Puzzle>, "id">) {
      const [_, changeId] = await updateRequest<Puzzle>(`/puzzles/${id}`, data);
      optimistic.set(changeId, { type: "puzzle", id, ...data });
      refresh();
    },
    async updatePuzzleOptimistic(id: number, data: Omit<Partial<Puzzle>, "id">) {
      const localId = optimisticCounter++;
      const delta: Optimistic = { type: "puzzle", id, ...data };
      optimistic.set(localId, delta);
      refresh();
      try {
        const [_, changeId] = await updateRequest<Puzzle>(`/puzzles/${id}`, data);
        optimistic.set(changeId, delta);
      } finally {
        optimistic.delete(localId);
        refresh();
      }
    },
    async deletePuzzle(id: number) {
      const [_, changeId] = await updateRequest<Puzzle>(
        `/puzzles/${id}`, { delete: true });
      optimistic.set(changeId, { type: "puzzle.delete", id });
      refresh();
    },
  };
  provide(key, state); // must run before first `await`

  const { data, error } = await useFetch<HomeResponse>("/api/home");
  if (!data.value) {
    throw error.value;
  }
  initialChangeId = data.value.change_id;
  data.value.puzzles.forEach((p) => _puzzles.set(p.id, p));
  data.value.rounds.forEach((r) => _rounds.set(r.id, r));
  onSettings(data.value.settings);

  refresh();
  return state;
};

export default function usePuzzles(): State {
  const state = inject(key);
  if (!state) {
    // Note: usePuzzles() will only work in a lower-level component than the
    // one where initializePuzzles() was called.
    throw new Error("Called usePuzzles() before initializing");
  }
  return state;
}
