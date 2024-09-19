#!/bin/bash

# This script is used to test the API of the application

# Check if jq is available
if ! [ -x "$(command -v jq)" ]; then
  echo 'Error: jq is not installed.' >&2
  exit 1
fi

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
API_CALL=$1
API_METHOD=${2:-GET}

# Check if the API call is provided
if [ -z "$API_CALL" ]; then
  echo 'Error: The API call is not provided.' >&2
  exit 1
fi

# Check if the application is running
if ! curl -s $API_URL/health > /dev/null; then
  echo 'Could not reach server at $API_URL, make sure the application is running, or ' >&2
  echo 'if API_HOST, API_PORT, and API_PROTOCOL environment variables, if provided.' >&2
  exit 1
fi

TOKEN=$(curl -s -X GET -u $API_USER:$API_PASSWORD $API_URL/api/v1/authenticate | jq -r '.token')

if [ -z "$TOKEN" ]; then
  echo 'Error: Could not authenticate with the API.' >&2
  exit 1
fi

case $API_METHOD in
  GET)
    curl -s -X GET $API_URL/api/v1/$API_CALL -H "Authorization: Bearer $TOKEN" | jq
    exit 0
    ;;
  POST | PATCH | PUT)
    CALL_BODY=${@:3}
    if [ -z "$CALL_BODY" ]; then
      curl -sv -X $API_METHOD $API_URL/api/v1/$API_CALL -H "Authorization: Bearer $TOKEN" 
      exit 0
    else
      if [ -f "$CALL_BODY" ]; then
        curl -sv -X $API_METHOD $API_URL/api/v1/$API_CALL -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "@$CALL_BODY" 
        exit 0
      else
        curl -sv -X $API_METHOD $API_URL/api/v1/$API_CALL -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$CALL_BODY" 
        exit 0
      fi
    fi
    ;;
  DELETE)
    curl -s -X DELETE $API_URL/api/v1/$API_CALL -H "Authorization: Bearer $TOKEN" | jq
    ;;
  *)
    echo 'Error: Invalid API method.' >&2
    exit 1
    ;;
esac
