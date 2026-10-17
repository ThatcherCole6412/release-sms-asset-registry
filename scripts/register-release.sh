#!/bin/sh
set -eu

curl --fail-with-body --request POST "http://localhost:8080/release/sms-assets" \
  --header "Content-Type: application/json" \
  --data '{"product":"ForgePipe","release_id":"2026-09-06.1","build_event":"succeeded","operation":"promote","diagnostic":"clear","signature":"ForgePipe","proof_url":"https://example.com/forgepipe"}'
