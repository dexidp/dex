# This file will do everything necessary to bring up a working Dex
# environment, connected to a Postgres DB and with a local and Google OIDC
# connector; When the script is completed, you will have three processes running
# in the background of your (bash) shell: an Dex Overlord, an Dex Worker,
# and the example app.
#
# It assumes you are in the root directory of the Dex project and that you
# have psql installed and running.
#
# USAGE:
#
# DEX_GOOGLE_CLIENT_ID=<<your_client_id>> DEX_GOOGLE_CLIENT_SECRET=<<your_client_secret>> && source  contrib/standup-db.sh
#
# NOTE: As you can see from above, this file is meant to be *sourced* not executed directly.

# Build components.
./build

# Set DB var
DEX_DB=dex_dev
DEX_DB_URL=postgres://localhost/$DEX_DB?sslmode=disable
export DEX_WORKER_DB_URL=$DEX_DB_URL

# Delete/create DB
dropdb $DEX_DB; createdb $DEX_DB


DEX_KEY_SECRET=$(dd if=/dev/random bs=1 count=32 2>/dev/null | base64)

# Start the overlord
export DEX_OVERLORD_DB_URL=$DEX_DB_URL
export DEX_OVERLORD_KEY_SECRETS=$DEX_KEY_SECRET
export DEX_OVERLORD_KEY_PERIOD=1h
./bin/dex-overlord &
echo "Waiting for overlord to start..."
until $(curl --output /dev/null --silent --fail http://localhost:5557/health); do
    printf '.'
    sleep 1
done

# Create a client 
eval "$(./bin/dexctl -db-url=$DEX_DB_URL new-client http://127.0.0.1:5555/callback)"

# Set up connectors
DEX_CONNECTORS_FILE=$(mktemp  /tmp/dex-conn.XXXXX)
DEX_GOOGLE_ISSUER_URL=https://accounts.google.com 
cat << EOF > $DEX_CONNECTORS_FILE
[
	{
		"type": "local",
		"id": "local"
	},
	{
		"type": "oidc",
		"id": "google",
		"issuerURL": "$DEX_GOOGLE_ISSUER_URL",
		"clientID": "$DEX_GOOGLE_CLIENT_ID",
		"clientSecret": "$DEX_GOOGLE_CLIENT_SECRET",
		"trustedEmailProvider": true
	}
]
EOF

./bin/dexctl -db-url=$DEX_DB_URL set-connector-configs $DEX_CONNECTORS_FILE


# Start the worker
export DEX_WORKER_DB_URL=$DEX_DB_URL
export DEX_WORKER_KEY_SECRETS=$DEX_KEY_SECRET
export DEX_WORKER_LOG_DEBUG=1
./bin/dex-worker &
echo "Waiting for worker to start..."
until $(curl --output /dev/null --silent --fail http://localhost:5556/health); do
    printf '.'
    sleep 1
done

# Start the app
./bin/example-app --client-id=$DEX_APP_CLIENT_ID --client-secret=$DEX_APP_CLIENT_SECRET --discovery=http://127.0.0.1:5556 &

# Create Admin User - the password is a hash of the word "password"
curl -X POST --data '{"email":"admin@example.com","password":"$2a$04$J54iz31fhYfXIRVglUMmpufY6TKf/vvwc9pv8zWog7X/LFrFfkNQe" }' http://127.0.0.1:5557/api/v1/admin

