#!/bin/bash

for index in $(seq 1000); do
    value=$(seq -w -s '' $index $(($index + 100000)))
    eval array$index=$value
done
