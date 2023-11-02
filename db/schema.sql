CREATE TABLE puzzles (
    id              INTEGER PRIMARY KEY,
    name            TEXT    NOT NULL,
    answer          TEXT    NOT NULL,
    round           INTEGER,
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
    reminder        DATETIME,

    FOREIGN KEY(round) REFERENCES rounds(id)
);

CREATE TABLE rounds (
    id              INTEGER PRIMARY KEY,
    name            TEXT    NOT NULL,
    emoji           TEXT    NOT NULL
)
