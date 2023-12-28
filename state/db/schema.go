package db

import _ "embed"

//go:embed schema.sql
var DDL string

func (r Round) HasDriveFolder() bool {
	return r.DriveFolder != "" && r.DriveFolder != "-"
}

func (r Round) HasDiscordCategory() bool {
	return r.DiscordCategory != "" && r.DiscordCategory != "-"
}
