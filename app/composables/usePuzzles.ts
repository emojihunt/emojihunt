// Reactive puzzle state (public interface).
//
// It's safe to destructure the immediate keys of this object.
//
type State = {
  connected: Ref<boolean>,
  active: Ref<boolean>,
  discordCallback: Ref<((m: DiscordMessage) => void) | undefined>;

  activity: Map<number, Map<string, boolean>>;
  settings: Settings;
  users: Map<string, User>;

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

type User = {
  username: string;
  avatarUrl: string;
};

type Optimistic = (
  ({ type: "puzzle"; } & Partial<Puzzle>) |
  ({ type: "round"; } & Partial<Round>) |
  ({ type: "puzzle.delete", id: number; }) |
  ({ type: "round.delete", id: number; })
) & { id: number; };

const key = Symbol() as InjectionKey<State>;

const updateRequest = async <T>(endpoint: string, params: any): Promise<[T, number]> => {
  let response;
  if (params.delete === true) {
    response = await formSubmit(endpoint, {}, "DELETE");
  } else {
    response = await formSubmit(endpoint, params);
  }
  if (response.status === 401) {
    window.location.reload();
  } else if (response.status !== 200) {
    throw createError({
      fatal: true,
      statusCode: response.status,
      data: response._data,
    });
  }
  const header = response.headers.get("X-Change-ID");
  if (!header) throw "Missing X-Change-ID header";
  return [await response._data, parseInt(header)];
};

const updateReactiveMap = <K, V>(m: Map<K, V>, k: K, v: V): void => {
  const existing = m.get(k);
  if (existing) Object.assign(existing, v);
  else m.set(k, { ...v });
};

const hydrateRound = (raw: Round, puzzles: Puzzle[]): AnnotatedRound => {
  const metas = puzzles.filter((p => p.meta));
  const complete = puzzles.length > 0 &&
    (metas.length === 0 ? puzzles : metas).filter((p => !p.answer)).length === 0;
  return {
    ...raw,
    anchor: raw.name.trim().toLowerCase().replaceAll(/[^A-Za-z0-9]+/g, "-"),
    complete,
    priority: !complete && !raw.special,
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
export async function initializePuzzles(pageId?: number): Promise<State> {
  if (inject(key, undefined)) {
    throw new Error("usePuzzles() may only be initialized once");
  }
  if (import.meta.server && !useCookie("session").value) {
    throw createError({
      message: "short-circuiting to login page",
      statusCode: 401,
    });
  }

  const activity = reactive(new Map<number, Map<string, boolean>>());
  const settings: Settings = reactive({
    discordGuild: "", hangingOut: "", huntName: "", huntURL: "",
    huntCredentials: "", logisticsURL: "", nextHunt: null,
  });
  const users = reactive(new Map<string, User>());

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
  let latestChangeId = 0;
  let searching = false;

  const refresh = () => { // clocks at around 4ms
    // First, materialize optimistically-applied updates.
    const localPuzzles = new Map(_puzzles);
    const localRounds = new Map(_rounds);
    for (const [_, entry] of [...optimistic.entries()].sort()) {
      switch (entry.type) {
        case "puzzle":
          let p = localPuzzles.get(entry.id);
          if (p) localPuzzles.set(entry.id, { ...p, ...entry });
          break;
        case "round":
          let r = localRounds.get(entry.id);
          if (r) localRounds.set(entry.id, { ...r, ...entry });
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
    puzzles.forEach((_, k) => localPuzzles.has(k) || puzzles.delete(k));

    const grouped = new Map<number, Puzzle[]>();
    for (const puzzle of puzzles.values()) {
      const g = grouped.get(puzzle.round);
      if (g) g.push(puzzle);
      else grouped.set(puzzle.round, [puzzle]);
    }
    for (const [_, puzzles] of grouped) {
      puzzles.sort((a, b) => {
        if (a.meta !== b.meta) return a.meta ? 1 : -1;
        const ra = parseTimestamp(a.reminder);
        const rb = parseTimestamp(b.reminder);
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

    localRounds.forEach((v, k) => updateReactiveMap(rounds, k,
      hydrateRound(v, grouped.get(v.id) || [])));
    rounds.forEach((_, k) => localRounds.has(k) || rounds.delete(k));

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

  const onSync = ({ change_id, kind, puzzle, round }: SyncMessage) => {
    if (change_id < latestChangeId) {
      return;
    } else if (change_id === latestChangeId) {
      console.log("Live sync caught up!");
      searching = false; // we're all caught up!
      return;
    } else if (searching) {// change_id > latestChangeId
      console.warn("Live worker got too far ahead. Reloading page...");
      window.location.reload();
      return;
    }
    for (const key of [...optimistic.keys()]) {
      if (key <= change_id) optimistic.delete(key);
    }
    latestChangeId = change_id;
    if (kind === "upsert") {
      if (puzzle) _puzzles.set(puzzle.id, puzzle);
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
    settings.nextHunt = parseTimestamp(msg.next_hunt);

    Object.entries(msg.voice_rooms).forEach(([id, raw]) => {
      // We expect the channel's emoji to go at the end
      const p = raw.split(" ");
      if ([...p[p.length - 1]!].length === 1) {
        const name = p.slice(0, p.length - 1).join(" ");
        const emoji = p[p.length - 1]!;
        voiceRooms.set(id, { id, name, emoji });
      } else {
        voiceRooms.set(id, { id, name: raw, emoji: "📻" });
      }
    });
    const ids = new Set<string>(Object.keys(msg.voice_rooms));
    voiceRooms.forEach((_, i) => !ids.has(i) && voiceRooms.delete(i));
  };
  const onActivity = (m: ActivityMessage) => {
    Object.entries(m).forEach(
      ([k, v]) => activity.set(parseInt(k), new Map(Object.entries(v)))
    );
    activity.forEach((_, k) => Object.hasOwn(m, k.toString()) || activity.delete(k));
  };
  const discordCallback = ref<(m: DiscordMessage) => void>();
  const onDiscord = (m: DiscordMessage) => discordCallback.value?.(m);
  const onUsers = (m: UsersMessage) => {
    if (m.replace) users.clear();
    Object.entries(m.users).forEach(
      ([k, [username, avatarUrl]]) => users.set(k, { username, avatarUrl })
    );
    m.delete?.forEach((k) => users.delete(k));
  };
  const [connected, active] = useAbly(pageId, onActivity, onDiscord, onSettings, onSync, onUsers);

  const state: State = {
    connected, active, discordCallback,
    activity, settings, users,
    puzzles, rounds, voiceRooms,
    puzzleCount, solvedPuzzleCount, ordering,
    async addRound(data: NewRound) {
      const [round, changeId] = await updateRequest<Round>("/rounds", data);
      if (changeId > latestChangeId) {
        optimistic.set(changeId, { type: "round", ...round });
      }
      refresh();
    },
    async updateRound(id: number, data: Omit<Partial<Round>, "id">) {
      const [_, changeId] = await updateRequest<Round>(`/rounds/${id}`, data);
      if (changeId > latestChangeId) {
        optimistic.set(changeId, { type: "round", id, ...data });
      }
      refresh();
    },
    async updateRoundOptimistic(id: number, data: Omit<Partial<Round>, "id">) {
      const localId = optimisticCounter++;
      const delta: Optimistic = { type: "round", id, ...data };
      optimistic.set(localId, delta);
      refresh();
      try {
        const [_, changeId] = await updateRequest<Round>(`/rounds/${id}`, data);
        if (changeId > latestChangeId) {
          optimistic.set(changeId, delta);
        }
      } finally {
        optimistic.delete(localId);
        refresh();
      }
    },
    async deleteRound(id: number) {
      const [_, changeId] = await updateRequest<Puzzle>(
        `/rounds/${id}`, { delete: true });
      if (changeId > latestChangeId) {
        optimistic.set(changeId, { type: "round.delete", id });
      }
      refresh();
    },
    async addPuzzle(data: NewPuzzle) {
      const [puzzle, changeId] = await updateRequest<Puzzle>("/puzzles", data);
      if (changeId > latestChangeId) {
        optimistic.set(changeId, { type: "puzzle", ...puzzle });
      }
      refresh();
    },
    async updatePuzzle(id: number, data: Omit<Partial<Puzzle>, "id">) {
      const [_, changeId] = await updateRequest<Puzzle>(`/puzzles/${id}`, data);
      if (changeId > latestChangeId) {
        optimistic.set(changeId, { type: "puzzle", id, ...data });
      }
      refresh();
    },
    async updatePuzzleOptimistic(id: number, data: Omit<Partial<Puzzle>, "id">) {
      const localId = optimisticCounter++;
      const delta: Optimistic = { type: "puzzle", id, ...data };
      optimistic.set(localId, delta);
      refresh();
      try {
        const [_, changeId] = await updateRequest<Puzzle>(`/puzzles/${id}`, data);
        if (changeId > latestChangeId) {
          optimistic.set(changeId, delta);
        }
      } finally {
        optimistic.delete(localId);
        refresh();
      }
    },
    async deletePuzzle(id: number) {
      const [_, changeId] = await updateRequest<Puzzle>(
        `/puzzles/${id}`, { delete: true });
      if (changeId > latestChangeId) {
        optimistic.set(changeId, { type: "puzzle.delete", id });
      }
      refresh();
    },
  };
  provide(key, state); // must run before first `await`

  const { data, error } = await useAPI<HomeResponse>("/home");
  if (!data.value) {
    throw error.value;
  }
  latestChangeId = data.value.change_id;
  if (latestChangeId > 0) {
    searching = true;
  }
  data.value.puzzles.forEach((p) => _puzzles.set(p.id, p));
  data.value.rounds.forEach((r) => _rounds.set(r.id, r));
  onSettings(data.value.settings);

  refresh();
  return state;
};

export default function usePuzzles(): State {
  const state = inject(key, undefined);
  if (!state) {
    // Note: usePuzzles() will only work in a lower-level component than the
    // one where initializePuzzles() was called.
    throw new Error("Called usePuzzles() before initializing");
  }
  return state;
}
