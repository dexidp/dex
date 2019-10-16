# CORS

CORS (cross-origin-resource-sharing) allows servers to specify not only who can access protected resources but also how those resources are accessed. For a more indepth explanation for CORS please refer to https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS

To successfully make a request to one of Dex's protected endpoints a CORS configuration will have to be set up.

Dex is built on top of Gorilla's router and is able to access the standard CORS package provided by Gorilla. This is accessed via the `handlers` package.
See `"github.com/gorilla/handlers"` for mor information on included functionality.

## Protected endpoints

Any requests to the following endpoints provided by Dex during runtime is subject to CORS policies

* /token
* /keys
* /userinfo
* /.well-known/openid-configuration


## Dex defaults

By default a Dex installation will turn off CORS if the `allowedOrigins` option is not set.

If CORS is turned on by setting the `allowedOrigins`. The following are the default values which will be set.
* allowedHeaders: "Accept", "Accept-Language", "Content-Language", "Origin"
* allowedMethods: "GET", "HEAD", "POST"
* exposedHeaders: {empty}
* ignoreOptions: false
* allowCredentials: false
* maxAge: default 0 seconds, max 600 seconds (10 minutes)
* optionsStatuscode: 200


## CORS configuration

CORS functionality can be setup via the addition of the following items in the yml configuration file under the object `web`

Note: You can choose to omit several of these configurations as they have defaults already. However if you require CORS, you must at least have `allowedOrigins` set. 

```
# Configuration for the HTTP endpoints.
web:
  http: 0.0.0.0:5556
  # Allow all origins, or use a comma seperated list, If not present CORS is turned off.
  allowedOrigins: ["*"]
  # Allow the following specific headers, comma seperated list Note that this is an APPEND operation, you will always get "Accept", "Accept-Language", "Content-Language", "Origin"
  allowedHeaders: ["custom-header-1", "custom-header-2"]
  # Allow the following specific methods, comma seperated list. Note that this is a REPLACE operation, you will need to explicitly list all the methods you need to allow.
  allowedMethods: ["PATCH", "DELETE"]
  # Exposed headers, these are the headers the server will not strip out. 
  exposedHeaders: ["custom-header-1", "custom-header-2"]
  # Ignores request for OPTIONS (default: false)
  ignoreOptions: true
  # Allows Credentals to be passed with the request (default: false)
  allowCredentials: true
  # Sets the maximum age in seconds between pre-flight requests for OPTIONS (default: 0 seconds, max: 600 seconds)
  maxAge: 10
  # Sets a custom status code for the OPTIONS response, default is 200
  optionsStatuscode: 204
  
  # Uncomment for HTTPS options.
  # https: 127.0.0.1:5554
  # tlsCert: /etc/dex/tls.crt
  # tlsKey: /etc/dex/tls.key
```
## Swaggerui support with CORS
Should you wish to test out Dex against a Swaggerui such as https://petstore.swagger.io then you must configure CORS on Dex to support the following:
 
 * allowedHeaders: ["api_key", "Authorization"]

Additionally your swaggerui defintion file must contain the following:

```json
"securityDefinitions": {
    "OAuth2": {
      "type": "oauth2",
      "flow": "accessCode",
      "authorizationUrl": "http://localhost:5556/dex/auth",
      "tokenUrl": "http://localhost:5556/dex/token",
      "scopes": {
        "openid": "OpenID Connect scope",
        "admin": "Grants read and write access to administrative information",
        "read": "Grants read access",
        "write": "Grants write access"
      }
    }
  }
```


For more information please refer: https://swagger.io/docs/open-source-tools/swagger-ui/usage/cors/ 