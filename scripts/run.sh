#!/bin/bash

docker compose build accounts
docker compose run -p 3000:3000 accounts