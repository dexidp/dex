# Running Examples

The quickest way to start experimenting with dex is to run a single dex-worker locally, with an in-process database, and then interact with it using the example programs in this directory.

## Build Everything and Start dex-worker

First, build the example webapp client and example CLI client.

```console
./build
```

We can start dex in local mode. The default values for `dex-worker` flags are set to load
some example objects which will be used in the next steps.

```console
./bin/dex-worker --no-db &
```

## Example Webapp Client

Build and run the example app webserver by pointing the discovery URL to local Dex, and
supplying the client information from `./static/fixtures/clients.json` into the flags.

```console
./bin/example-app \
	--client-id=example-app \
	--client-secret=ZXhhbXBsZS1hcHAtc2VjcmV0 \
	--discovery=http://127.0.0.1:5556/dex
```

Visit [http://localhost:5555](http://localhost:5555) in your browser and click "login" link.
Next click "Login with Email" and enter the sample credentials from `static/fixtures/connectors.json`:

* email: `elroy77@example.com`
* password: `bones`

The example app will dump out details of the JWT issued by Dex which means that authentication was successful and the application has authenticated you as a valid user.
You can play with adding additional users in connectors.json and users.json.

## Example CLI Client

The example CLI will start, connect to the Dex instance to gather discovery information, listen on `localhost:8000`, and then acquire a client credentials JWT and print it out.

```console
./bin/example-cli \
	--client-id example-cli \
	--client-secret ZXhhbXBsZS1jbGktc2VjcmV0 \
	--discovery=http://127.0.0.1:5556/dex
```
