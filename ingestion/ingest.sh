#!/usr/bin/env bash
docker cp ingest.js sky-mongo:/
docker exec sky-mongo mongosh -u root -p rootpassword --authenticationDatabase admin ingest.js
