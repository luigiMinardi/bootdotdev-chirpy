#!/usr/bin/env bash
cd ./sql/schema/
goose postgres "postgres://lug:@localhost:5432/chirpy" up
