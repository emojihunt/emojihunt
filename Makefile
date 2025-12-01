.PHONY: all huntbot live app

all: huntbot live app

huntbot:
	fly deploy

live:
	fly deploy --config live/fly.toml

app:
	fly deploy app
