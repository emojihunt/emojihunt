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

## Google Drive Setup

This bot uses default Google credentials to authenticate to Google. This works
magically in Google Cloud, but requires some effort to make it work locally. To
do so, you will need to create a service account, and download the keypair for
it. Then, you will need to export the path to the file (on Linux). For other
operating systems, see the
[official Google docs](https://cloud.google.com/docs/authentication/production#passing_variable).

The service account will also need access to the sheet. Either the sheet can be
public to the world, or the sheet can be shared with the service account email
address.
