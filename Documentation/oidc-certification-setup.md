# OpenID Connect Provider Certification

The OpenID Foundation provides a set of [conformance test profiles][oidc-conf-profiles] that test both Relying Party and OpenID Provider (OP) OpenID Connect implementations. Upon submission of [results][oidc-result-submission] and an affirmative response, the affirmed OP will be listed as a [certified OP][oidc-certified-ops] on the OpenID Connect website and allowed to use the [certification mark][oidc-cert-mark] according to the certification [terms and conditions][oidc-terms-conds], section 3(d).

## Basic OpenID Provider Tests

Dex is an OP that strives to implement the [mandatory set][oidc-core-spec-mandatory] of OpenID Connect features, and can be tested against the Basic OpenID Provider profile ([profile outline][oidc-conf-profiles], section 2.1.1). These tests ensure that all features required by a [basic client][oidc-basic-client-spec] work as expected.

Features are currently under development to fully comply with the Basic profile, as dex currently does not. The following issues track our progress:

Issue number | Relates to
:---: | :---:
[\#376][dex-issue-376] | userinfo_endpoint
[\#1052][dex-issue-1052] | auth_time

[dex-issue-376]: https://github.com/dexidp/dex/issues/376
[dex-issue-1052]: https://github.com/dexidp/dex/issues/1052

### Setup

There are two ways to set up an OpenID test instance:
1. Configure a test instance provided by The OpenID Foundation by following [instructions][oidc-test-config] on their website.
1. Download their test runner from [GitHub][oidc-github] and follow the instructions in the [README][oidc-github-readme].
    * Requires `docker` and `docker-compose`

Configuration is essentially the same for either type of OpenID test instance. We will proceed with option 1 in this example, and set up an [AWS EC2 instance][aws-ec2-instance] to deploy dex:
* Create an [AWS EC2 instance][aws-ec2-quick-start] and connect to your instance using [SSH][aws-ec2-ssh].
* Install [dex][dex-install].
* Ensure whatever port dex is listening on (usually 5556) is open to ingress traffic in your security group configuration.
* In this example the public DNS name, automatically assigned to each internet-facing EC2 instance, is **my-test-ec2-server.com**. You can find your instances' in the AWS EC2 management console.

### Configuring an OpenID test instance

1. Navigate to [https://op.certification.openid.net:60000][oidc-test-start].
1. Click 'New' configuration.
1. Input your issuer url: `http://my-test-ec2-server.com:5556/dex`.
1. Select `code` as the response type.
1. Click 'Create' to further configure your OpenID instance.
1. On the next page, copy and paste the `redirect_uris` into the `redirectURIs` config field (see below).
1. At this point we can run dex, as we have all the information necessary to create a config file (`oidc-config.yaml` in this example):
    ```yaml
    issuer: http://my-test-ec2-server.com:5556/dex
    storage:
      type: sqlite3
      config:
        file: examples/dex.db
    web:
      http: 0.0.0.0:5556
    oauth2:
      skipApprovalScreen: true
    staticClients:
    - id: example-app
      redirectURIs:
      - 'https://op.certification.openid.net:${OPENID_SERVER_PORT}/authz_cb'
      name: 'Example App'
      secret: ZXhhbXBsZS1hcHAtc2VjcmV0
    connectors:
    - type: mockCallback
      id: mock
      name: Example
    ```
    * Substitute `OPENID_SERVER_PORT` for your OpenID test instance port number, assigned after configuring that instance.
    * Set the `oauth2` field `skipApprovalScreen: true` to automate some clicking.
1. Run dex:
    ```bash
    $ ./bin/dex serve oidc-config.yaml
    time="2017-08-25T06:34:57Z" level=info msg="config issuer: http://my-test-ec2-server.com:5556/dex"
    ...
    ```
1. Input `client_id` and `client_secret` from your config file.
    * The `id` and `secret` used here are from the example config file [`staticClients` field](../examples/config-dev.yaml#L50-L55).
1. Use data returned by the `GET /.well-known/openid-configuration` API call to fill in the rest of the configuration forms:
    ```bash
    [home@localhost ~]$ curl http://my-test-ec2-server.com:5556/dex/.well-known/openid-configuration
    {
      "issuer": "http://my-test-ec2-server.com:5556/dex",
      "authorization_endpoint": "http://my-test-ec2-server.com:5556/dex/auth",
      "token_endpoint": "http://my-test-ec2-server.com:5556/dex/token",
      "jwks_uri": "http://my-test-ec2-server.com:5556/dex/keys",
      "response_types_supported": [
        "code"
      ],
      "subject_types_supported": [
        "public"
      ],
      "id_token_signing_alg_values_supported": [
        "RS256"
      ],
      "scopes_supported": [
        "openid",
        "email",
        "groups",
        "profile",
        "offline_access"
      ],
      "token_endpoint_auth_methods_supported": [
        "client_secret_basic"
      ],
      "claims_supported": [
        "aud",
        "email",
        "email_verified",
        "exp",
        "iat",
        "iss",
        "locale",
        "name",
        "sub"
      ]
    }
    ```
    * Fill in all configuration information that the `/.well-known/openid-configuration` endpoint returns, althgouh this is not strictly necessary. We should give the test cases as much information about dex's OP implementation as possible.
1. Press the 'Save and Start' button to start your OpenID test instance.
1. Follow the provided link.
1. Run through each test case, following all instructions given by individual cases.
    * In order to pass certain cases, screenshots of OP responses might be required.

## Results and Submission

Dex does not fully pass the Basic profile test suite yet. The following table contains the current state of test results.

Test case ID | Result type | Cause | Relates to
--- | --- | --- | ---
OP-Response-Missing | Incomplete | Expected |
OP-Response-code | Succeeded | |
OP-Response-form_post | Succeeded | |
OP-IDToken-C-Signature | Succeeded | |
OP-ClientAuth-Basic-Static | Succeeded | |
OP-ClientAuth-SecretPost-Static | Warning | Unsupported | client_secret_post
OP-Token-refresh | Incomplete | Unsupported | userinfo_endpoint
OP-UserInfo-Body | Incomplete | Unsupported | userinfo_endpoint
OP-UserInfo-Endpoint | Incomplete | Unsupported | userinfo_endpoint
OP-UserInfo-Header | Incomplete | Unsupported | userinfo_endpoint
OP-claims-essential | Incomplete | Unsupported | userinfo_endpoint
OP-display-page | Succeeded | |
OP-display-popup | Succeeded | |
OP-nonce-NoReq-code | Succeeded | |
OP-nonce-code | Succeeded | |
OP-prompt-login | Succeeded | |
OP-prompt-none-LoggedIn | Succeeded | |
OP-prompt-none-NotLoggedIn | Incomplete | Error expected
OP-redirect_uri-NotReg | Incomplete | Requires screenshot
OP-scope-All | Incomplete | Unsupported | address, phone
OP-scope-address | Incomplete | Unsupported | address
OP-scope-email | Incomplete | Unsupported | userinfo_endpoint
OP-scope-phone | Incomplete | Unsupported | phone
OP-scope-profile | Incomplete | Unsupported | userinfo_endpoint
OP-Req-NotUnderstood | Succeeded | |
OP-Req-acr_values | Warning | No acr value | id_token
OP-Req-claims_locales | Incomplete | Unsupported | userinfo_endpoint
OP-Req-id_token_hint | Succeeded | |
OP-Req-login_hint | Incomplete | Missing configuration field | login_hint
OP-Req-max_age=1 | Failed | Missing configuration field | auth_time
OP-Req-max_age=10000 | Failed | Missing configuration field | auth_time
OP-Req-ui_locales | Succeeded | |
OP-OAuth-2nd | Warning | Unexpected error response | invalid_request
OP-OAuth-2nd-30s | Warning | Unexpected error response | invalid_request
OP-OAuth-2nd-Revokes | Incomplete | Unsupported | userinfo_endpoint

Once all test cases pass, submit your results by following instructions listed [on the website][oidc-result-submission].

[dex-install]: https://github.com/dexidp/dex/blob/master/Documentation/getting-started.md#building-the-dex-binary
[aws-ec2-instance]: http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/concepts.htmlSSH
[aws-ec2-ssh]: http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AccessingInstancesLinux.html
[aws-ec2-quick-start]: http://docs.aws.amazon.com/quickstarts/latest/vmlaunch/step-1-launch-instance.html
[oidc-core-spec-mandatory]: http://openid.net/specs/openid-connect-core-1_0.html#ServerMTI
[oidc-basic-client-spec]: http://openid.net/specs/openid-connect-basic-1_0.html
[oidc-conf-profiles]: http://openid.net/wordpress-content/uploads/2016/12/OpenID-Connect-Conformance-Profiles.pdf
[oidc-test-config]: http://openid.net/certification/testing/
[oidc-test-start]: https://op.certification.openid.net:60000
[oidc-result-submission]: http://openid.net/certification/submission/
[oidc-cert-mark]: http://openid.net/certification/mark/
[oidc-certified-ops]: http://openid.net/developers/certified/
[oidc-terms-conds]: http://openid.net/wordpress-content/uploads/2015/03/OpenID-Certification-Terms-and-Conditions.pdf
[oidc-github]: https://github.com/openid-certification/oidctest
[oidc-github-readme]: https://github.com/openid-certification/oidctest/blob/master/README.md
