#!/bin/bash

# Number of shards (same as NUM_SHARDS in launch.sh)
NUM_SHARDS=3

# Loop over each shard and populate
for i in $(seq 0 $((NUM_SHARDS-1)))
do
    SHARD_NAME="shard-$i"
    SHARD_ADDR="127.0.0.$((2*i+2)):8080"

    echo "Populating $SHARD_NAME at $SHARD_ADDR"

    # Insert 10000 random keys for this shard
    for j in {1..10000}; do
        KEY="key-$RANDOM"
        VALUE="value-$RANDOM"
        
        # Use curl to insert key-value pair
        curl -X POST "http://$SHARD_ADDR/set" -d "{\"key\": \"$KEY\", \"value\": \"$VALUE\"}" -H "Content-Type: application/json" > /dev/null &
    done

    wait # Ensure all requests are finished for this shard before continuing to next
done
