#!/usr/bin/env bash
set -euo pipefail

BASE_URL=http://localhost:8080/actuator
for actuator in $(curl --fail -s $BASE_URL) ; do
  URL=$BASE_URL/$actuator
  CURRENT_VALUE=$(curl --fail -s "$URL")
  echo "$actuator=$(curl --fail -s -X POST "$URL" -d "$CURRENT_VALUE")"
done