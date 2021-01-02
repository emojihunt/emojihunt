# emojihunt

This repo contains the technology used by the Emoji Hunt MIT Mystery Hunt team.
This consists of discord bots written in [Go](https://golang.org/), and may in
the future involve tools to sync our Google sheets.

In addition to this tech, we also use a Google sheets to solve puzzles.

## Discord Bot permissions model

There are 3 parts to the permissisons of the discord bot in discord\_bots.

1.  The bot should be part of a Discord application.
1.  Once registered, there will be a token for the bot. That must be passed
    with the `--discord_token` flag, including the "Bot " prefix.
1.  The bot should be a registered user in the discord server.
1.  In the future, the bot will need Google Drive permssions. The bot does not
    use end user credentials, and so should only be used for the Emoji Hunt
    discord server.
