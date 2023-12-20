CREATE TABLE puzzles (
    id              INTEGER PRIMARY KEY,
    name            TEXT    NOT NULL,
    answer          TEXT    NOT NULL,
    round           INTEGER NOT NULL,
    status          TEXT    NOT NULL,

    note            TEXT    NOT NULL,
    location        TEXT    NOT NULL,

    puzzle_url      TEXT    NOT NULL,
    spreadsheet_id  TEXT    NOT NULL,
    discord_channel TEXT    NOT NULL,

    meta            BOOLEAN NOT NULL,
    archived        BOOLEAN NOT NULL,

    voice_room      TEXT    NOT NULL,
    reminder        DATETIME    NOT NULL,

    FOREIGN KEY (round) REFERENCES rounds(id),
    CONSTRAINT uc_name_rd   UNIQUE(name, round)
);

CREATE TABLE rounds (
    id              INTEGER PRIMARY KEY,
    name            TEXT    NOT NULL,
    emoji           TEXT    NOT NULL,
    hue             INTEGER NOT NULL,
    special         BOOLEAN NOT NULL,

    CONSTRAINT uc_name      UNIQUE(name),
    CONSTRAINT uc_emoji     UNIQUE(emoji)
);

CREATE TABLE state (
    id                      INTEGER PRIMARY KEY,
    data                    BLOB
);
