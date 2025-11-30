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
    voice_room      TEXT    NOT NULL,
    reminder        DATETIME NOT NULL,

    FOREIGN KEY (round) REFERENCES rounds(id),
    CONSTRAINT uc_name_rd UNIQUE(name COLLATE nocase, round)
);

CREATE TABLE rounds (
    id              INTEGER PRIMARY KEY,
    name            TEXT    NOT NULL,
    emoji           TEXT    NOT NULL,
    hue             INTEGER NOT NULL,

    sort            INTEGER NOT NULL,
    special         BOOLEAN NOT NULL,

    drive_folder    TEXT    NOT NULL,
    discord_category TEXT   NOT NULL,

    CONSTRAINT uc_name  UNIQUE(name),
    CONSTRAINT uc_emoji UNIQUE(emoji)
);

CREATE TABLE changelog (
    id              INTEGER PRIMARY KEY,
    kind            TEXT    NOT NULL,
    puzzle          BLOB,
    round           BLOB
);

CREATE TABLE settings (
    key             TEXT    PRIMARY KEY,
    value           BLOB
);

CREATE TABLE discovered_puzzles (
    id              INTEGER PRIMARY KEY,
    puzzle_url      TEXT    NOT NULL,
    name            TEXT    NOT NULL,

    -- only set if puzzle is awaiting round creation
    discovered_round INTEGER,
    FOREIGN KEY (discovered_round) REFERENCES discovered_rounds(id)
);

CREATE TABLE discovered_rounds (
    id              INTEGER PRIMARY KEY,
    name            TEXT    NOT NULL,
    message_id      TEXT    NOT NULL,
    notified_at     DATETIME NOT NULL,
    created_as      INTEGER NOT NULL
);
