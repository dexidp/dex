# Running Examples

The quickest way to start experimenting with dex is to run a single dex-worker locally, with an in-process database, and then interact with it using the example programs in this directory.

## Build Everything and Start dex-worker

First, build the example webapp client and example CLI client.

```
./build
```

Now copy the example configurations into place to get dex configured.
You can customize these later but the defaults should work fine.

```
cp static/fixtures/connectors.json.sample static/fixtures/connectors.json
cp static/fixtures/users.json.sample static/fixtures/users.json
cp static/fixtures/emailer.json.sample static/fixtures/emailer.json
```

With `dex-worker` configuration in place we can start dex in local mode.

```
./bin/dex-worker --no-db &
```

## Example Webapp Client

Build and run the example app webserver by pointing the discovery URL to local Dex, and 
supplying the client information from `./static/fixtures/clients.json` into the flags.

```
./bin/example-app \
	--client-id=example-app \
	--client-secret=example-app-secret \
	--discovery=http://127.0.0.1:5556
```

Visit [http://localhost:5555](http://localhost:5555) in your browser and click "login" link.
Next click "Login with Local" and enter the sample credentials from `static/fixtures/connectors.json`:

```
email: elroy77@example.com
password: bones
```

The example app will dump out details of the JWT issued by Dex which means that authentication was successful and the application has authenticated you as a valid user.
You can play with adding additional users in connectors.json and users.json.

## Example CLI Client

The example CLI will start, connect to the Dex instance to gather discovery information, listen on `localhost:8000`, and then acquire a client credentials JWT and print it out.

```
./bin/example-cli \
	--client-id example-cli
	--client-secret examplie-cli-secret
	--discovery=http://127.0.0.1:5556
```
