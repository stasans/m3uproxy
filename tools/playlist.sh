#!/bin/bash


# Check if curl is available
if ! [ -x "$(command -v curl)" ]; then
  echo 'Error: curl is not installed.' >&2
  exit 1
fi

API_HOST=${API_HOST:-localhost}
API_PORT=${API_PORT:-8080}
API_PROTOCOL=${API_PROTOCOL:-http}
API_URL="$API_PROTOCOL://$API_HOST:$API_PORT"
API_USER=${API_USER:-admin}
API_PASSWORD=${API_PASSWORD:-admin}

OUT=output.m3u
curl -s -X GET -u $API_USER:$API_PASSWORD $API_URL/channels.m3u > $OUT
echo $OUT