CREATE TABLE puzzles (
    id              INTEGER PRIMARY KEY,
    name            TEXT    NOT NULL,
    answer          TEXT    NOT NULL,
    rounds          TEXT    NOT NULL,
    status          TEXT    NOT NULL,

    description     TEXT    NOT NULL,
    location        TEXT    NOT NULL,

    puzzle_url      TEXT    NOT NULL,
    spreadsheet_id  TEXT    NOT NULL,
    discord_channel TEXT    NOT NULL,

    original_url    TEXT    NOT NULL,
    name_override   TEXT    NOT NULL,
    archived        BOOLEAN NOT NULL,

    voice_room      TEXT    NOT NULL,
    reminder        DATETIME
);

CREATE TABLE rounds (
    id              INTEGER PRIMARY KEY,
    name            TEXT    NOT NULL,
    emoji           TEXT    NOT NULL,

    CONSTRAINT uc_name      UNIQUE(name),
    CONSTRAINT uc_emoji     UNIQUE(emoji)
);

CREATE TABLE state (
    id                      INTEGER PRIMARY KEY,
    data                    BLOB
);
