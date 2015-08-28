Running Examples
===

The quickest way to start experimenting with dex is to run a single dex-worker
locally, with an in-process database, and then interacting with it using the
example programs in this directory.


## Build Everything and Start dex-worker

This section is required for both the Example App and the Example CLI. 

1. Build everything:
   ```
   ./build
   ```
   
1. Copy the various example configurations.
    ```
    cp static/fixtures/connectors.json.sample static/fixtures/connectors.json
    cp static/fixtures/users.json.sample static/fixtures/users.json
    cp static/fixtures/emailer.json.sample static/fixtures/emailer.json
    ```
    
1. Run dex_worker in local mode.
    ```
    ./bin/dex-worker --no-db &
    ```


## Example App

1. Build and run example app webserver, pointing the discovery URL to local Dex, and 
supplying the client information from `./static/fixtures/clients.json` into the flags.
   ```
   ./bin/example-app --client-id=XXX --client-secret=secrete --discovery=http://127.0.0.1:5556 &
   ```

1. Navigate browser to `http://localhost:5555` and click "login" link
1. Click "Login with Local"
1. Enter in sample credentials from `static/fixtures/connectors.json`:
   ```
   email: elroy77@example.com
   password: bones
   ```
1. Observe user information in example app.
  
## Example CLI
*TODO*
