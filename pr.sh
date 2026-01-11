#!/bin/bash

PROXY_ADDR="127.0.0.1:8443"
# Extract hosts from config.json
HOSTS=$(grep -oP '"\K[^"]+(?=":)' config.json | grep ".org")

echo "🚀 Starting asynchronous test against $(echo "$HOSTS" | wc -l) hosts..."

for host in $HOSTS; do
    (
        # -k: ignore self-signed cert
        # --resolve: force the domain to point to your local proxy
        # -s: silent
        # -o /dev/null: don't print the response body
        curl -s -k --resolve "$host:8443:127.0.0.1" "https://$host:8443" \
             --connect-timeout 2 --max-time 5 -o /dev/null
        
        if [ $? -eq 0 ] || [ $? -eq 35 ]; then
            echo "✅ Finished: $host"
        else
            echo "❌ Failed: $host"
        fi
    ) & # The '&' makes it run in the background (asynchronously)
done

wait # Wait for all background tests to finish
echo "🏁 All tests dispatched."
