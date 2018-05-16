#!/bin/sh
set -e -x

# Add a token from https://github.com/settings/tokens to `github.oauth2_token`.
# This is needed to get a enhanced rate limit

go run build.go `cat github.oauth2_token`

COMMITS_PER_SECOND=25
ARGS="\
	-1920x1080\
	--background-image logo.png\
	--user-scale 4\
	--dir-colour FFFF00\
	--font-size 16\
	--seconds-per-day `expr 86400 / $COMMITS_PER_SECOND`\
	--key\
	--max-user-speed 1400\
	--user-friction .2\
	--hide date,filenames,mouse,progress\
	--bloom-intensity .6\
	--file-idle-time 0\
	--user-image-dir users"

gource $ARGS built_log.log

#gource $ARGS -o - --output-framerate 60 built_log.log | ffmpeg\
# -y -r 60 -f image2pipe -c:v ppm -i -\
# -c:v libx264 -threads 8 -preset medium -qp 18 runelite.mp4