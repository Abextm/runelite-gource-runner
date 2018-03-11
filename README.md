Scripts to run gource on the runelite repos

Deps:
 - sh
 - gource
 - go
 - git

I copy the font from `repos\RuneLite\runelite-client\src\main\resources\net\runelite\client\ui\runescape.ttf` over
`C:\Program Files\Gource\data\fonts\FreeSans.ttf` for that extra osrs feel, though it looks kindof bad, because you can't
change the gource's directory font size

Add a token from https://github.com/settings/tokens to `github.oauth2_token`.
This is needed to reduce the rate limit

Run `run.sh`. Uncomment the second gource command to generate a mp4