package main

import (
    "html/template"
    "log"
    "net/http"
)

var indexTmpl = template.Must(template.New("index.html").Parse(`<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <!-- The above 3 meta tags *must* come first in the head; any other head content must come *after* these tags -->
    <title>KubeAuth Login</title>

    <!-- Bootstrap -->
        <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css">
        <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap-theme.min.css">
        <script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/js/bootstrap.min.js"></script>

    <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/font-awesome/4.7.0/css/font-awesome.min.css">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/bootstrap-social/5.1.1/bootstrap-social.min.css">
    <!-- HTML5 shim and Respond.js for IE8 support of HTML5 elements and media queries -->
    <!-- WARNING: Respond.js doesn't work if you view the page via file:// -->
    <!--[if lt IE 9]>
      <script src="https://oss.maxcdn.com/html5shiv/3.7.3/html5shiv.min.js"></script>
      <script src="https://oss.maxcdn.com/respond/1.4.2/respond.min.js"></script>
    <![endif]-->
    <style>
      body {
        padding: 40px;
      }
    </style>
  </head>
  <body>
    <div class="row">
      <div class="col-xs-12 col-sm-4 col-md-3">
            <form action="/login" method="post">
              <div style="display:none;">
                   <p>
                       Authenticate for:<input type="text" name="cross_client" placeholder="list of client-ids" value="kubeauth">
                   </p>
                   <p>
                       Extra scopes:<input type="text" name="extra_scopes" placeholder="list of scopes" value="groups">
                   </p>
                 <p>
                     Request offline access:<input type="checkbox" name="offline_access" value="yes" checked>
                   </p>
               </div>
                 <!-- <input type="submit" value="Login"> -->
           <button type="submit" class="btn btn-block btn-social btn-github">
             <span class="fa fa-github"></span>
             Sign in with GitHub
           </button>
            </form>
      </div>
    </div>
  </body>
</html>`))

func renderIndex(w http.ResponseWriter) {
    renderTemplate(w, indexTmpl, nil)
}

type tokenTmplData struct {
    IDToken      string
    RefreshToken string
    RedirectURL  string
    Claims       string
}

var tokenTmpl = template.Must(template.New("token.html").Parse(`<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <!-- The above 3 meta tags *must* come first in the head; any other head content must come *after* these tags -->
    <title>KubeAuth Login</title>

    <!-- Bootstrap -->
        <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap.min.css">
        <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/css/bootstrap-theme.min.css">
        <script src="https://maxcdn.bootstrapcdn.com/bootstrap/3.3.7/js/bootstrap.min.js"></script>

    <link rel="stylesheet" href="https://maxcdn.bootstrapcdn.com/font-awesome/4.7.0/css/font-awesome.min.css">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/bootstrap-social/5.1.1/bootstrap-social.min.css">
    <!-- HTML5 shim and Respond.js for IE8 support of HTML5 elements and media queries -->
    <!-- WARNING: Respond.js doesn't work if you view the page via file:// -->
    <!--[if lt IE 9]>
      <script src="https://oss.maxcdn.com/html5shiv/3.7.3/html5shiv.min.js"></script>
      <script src="https://oss.maxcdn.com/respond/1.4.2/respond.min.js"></script>
    <![endif]-->
    <style>
      body {
        padding: 40px;
      }
            .pre-y-scrollable {
        word-wrap: break-word;
        white-space: pre-wrap;
        max-height: 400px;
      }
    </style>
  </head>
  <body>
    <div class="row">
      <div class="col-xs-12 col-sm-6 col-md-6">
        <div class="panel panel-success">
          <div class="panel-heading">kubeconfig</div>
          <div class="panel-body">
            <pre class="pre-y-scrollable">apiVersion: v1
kind: Config
preferences:
  colors: true
users:
  - name: github-oauth
    user:
      auth-provider:
        config:
          client-id: kubeauth
          client-secret: 2948b24a-fc8e-11e6-9ca8-731683843ad4
          id-token: {{ .IDToken }}
          idp-issuer-url: https://dex.hurley.hbo.com
          refresh-token: {{ .RefreshToken }}
        name: oidc
clusters:
  - cluster:
      certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURHakNDQWdLZ0F3SUJBZ0lKQVBIOFhpa05Xb2hSTUEwR0NTcUdTSWIzRFFFQkJRVUFNQkl4RURBT0JnTlYKQkFNVEIydDFZbVV0WTJFd0hoY05NVFl3TXpJeE1qQXpNakEzV2hjTk5ETXdPREEzTWpBek1qQTNXakFTTVJBdwpEZ1lEVlFRREV3ZHJkV0psTFdOaE1JSUJJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBCnM3cmVrR29OUE13VEttMkdtdHFmMnhILzUzNS9FT3pEa0gxOStsWnRRZytsQlZuN3RteldwQlZ4NEtmQVZwb1gKcTRRQldiNUNkeDdSUnJkV0ViVEFyVjRpZG1EMmRqU0krWk01TEY0UmVxalhQbURFWWxLRDJQcTVsajR1K0FBMgpYS1VVRXBVZEVOR3cwbXFpSGxDS1Y5dnhUdGlGcmtkSXZ2UXI2QkgyNk5UNk1qY25ZT2RhTVBodFcvOU8zc29tCk5VaEFNVmZ2OC9qaHhaL0trTTlLb0VRM21GbnIzLzh5QzRmbjhjZUZuQ1NjenA5eE96MU1DZjgvOVhFR0hTL0IKZXhZblRQckdTY0R1Z0lQZFBGK1RidzFkbEF2K2QyeFppd1Q3VVQ2QjdjcVovZmZ5enRHeFF6TDhFSy9mYWdGMApzNGF2RlMvU3hwMVNPamhjTGJSNWhRSURBUUFCbzNNd2NUQWRCZ05WSFE0RUZnUVVtSFpZd21oMFlhZU4yK0V1CkliQmVDbUgxUk13d1FnWURWUjBqQkRzd09ZQVVtSFpZd21oMFlhZU4yK0V1SWJCZUNtSDFSTXloRnFRVU1CSXgKRURBT0JnTlZCQU1UQjJ0MVltVXRZMkdDQ1FEeC9GNHBEVnFJVVRBTUJnTlZIUk1FQlRBREFRSC9NQTBHQ1NxRwpTSWIzRFFFQkJRVUFBNElCQVFDRGp6WXdMOXBBU2xSUHRPNU0xeEREUDF0MzdmR3NKdWU0RGR0TnlJS0gvS0VaCnVIKzlaQnRSekoyZ3JtRHVQK1ArWjRrTHk0S2xPYm93R2VvU0VpVEhkU2RDMFdjeE1nbGhpa1RlQTEzMTZVZHgKREVua0x1YTQwbmJpUDJvTUdVbmkzU3VkWWNkbVlWa3J1NWFyYVV2elRCOTJTOHJ6amxXblZDZFArVE43SXdIdgp4emtPQUNvWlViQlBPaDJDd2hicWhZTnRkd09VL3hXQmw0bkJJL2dzTy9sOWQyRVpyN2pEdVlZcmpUNXFpa2FvCnZHVFpvS1FZMmgwYnRaeVlOTWxXNVI0YkFRZmJjNzArTWpDYjFPeGhRUVdXcTQ3aTZkYWt5UjRSaU1oZUJ3QVkKbDhwdEhLZnJWVnA2TUFmWDFQa1R2UXJrNmxPakYrbDdSdVNOMk1lcQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
      server: https://k8s-hurley-master-us-west-2.api.hbo.com
    name: hurley-production-us-west-2
  - cluster:
      certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURHakNDQWdLZ0F3SUJBZ0lKQU1RTy9Zdm5qcFplTUEwR0NTcUdTSWIzRFFFQkJRVUFNQkl4RURBT0JnTlYKQkFNVEIydDFZbVV0WTJFd0hoY05NVFl3TXpJek1UUXlOak0zV2hjTk5ETXdPREE1TVRReU5qTTNXakFTTVJBdwpEZ1lEVlFRREV3ZHJkV0psTFdOaE1JSUJJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBCjU3dU16citpdXJ4dngvY1BZZm1tckJYUkxaYy9QMGJTWWVQWVFHVEdCelMzTmF1MU5BaTlFdGhOb1M1TDgrODgKV2IxYjh4UTF1WWJUTlhDUm0vRmFOMW5LNW5scGw4VFdpRm9nUGZwbVNhTjNkWkEzU0ZlSVI1ZXRRMHNLOUY4Kwo1M0l0T2VmeEYvU3dCNFlWVzFrb0VvN1dWcXQ2WVFBdVNyOWw0cHFXS0tSMjJPWmtJVjdnRUFqWXFTNWpObUlvCkRKVDVlZ1YyelJMU1ZIaVNRRFRvRTFEaWlMajlUOG95KzZYOUlWejExSGJORm04MG4remIzU21TOUVuc2hmRm0KSlc1TnRRVHY5Nldhd1RNN3NkS2pIaDhnZFFtWW4wUjNtdGRCR0s4SWZlV0FqQU5NVmJMdC9Sb045ci90NmFtbApIZlY2UVFORmNtSi9xNk10dDkvdVF3SURBUUFCbzNNd2NUQWRCZ05WSFE0RUZnUVVKYkovSDdlQTdBQ0l4M0xUCiszV0VCK3E4VE9Nd1FnWURWUjBqQkRzd09ZQVVKYkovSDdlQTdBQ0l4M0xUKzNXRUIrcThUT09oRnFRVU1CSXgKRURBT0JnTlZCQU1UQjJ0MVltVXRZMkdDQ1FERUR2Mkw1NDZXWGpBTUJnTlZIUk1FQlRBREFRSC9NQTBHQ1NxRwpTSWIzRFFFQkJRVUFBNElCQVFBMFdRcFZMWllWcVBWdllxM0xFQ25TUjBUcFovbW9sMGZQK0ZKVmRjMGM4bVZiCkxrbXdsUWZZRFd5blVyQWxOdVJrSlJxa3JVYm5acnVhdFU0bTEyZkFNVG5jZWcxaGFia3Vzc3Z3YzVkQUN5Q2UKSGp1R0hoUk84TThyQzIwMkUrTHZabEovbnVLVExEVTFMaGphZThBM1lDUWIxMFY1NFp4YVJUaHdFU2dhcm5lSgphdmEzdWs5VVd2SmF1MW1ydllTZnF6TGcwZnAwdDVIdzVQcHp5aUw3WHFQOUZOY3ZRb3FaNkVGcmFneFlHYVNMCmJDSTFWeVFZOEo2d3BRZ0lXQkxDbWFlbmk1K0syZGtPdm1xU0tRVUw4ZHRETGVBQ3Z3WUJSMFdrYm1XRVRnSVEKRC93T21VdjlOQWJkNW1pL2ZZanpyVzFuUFY2VkYxQnV6ckx0bWQrNAotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
      server: https://k8s-hurley-master-us-east-1.api.hbo.com
    name: hurley-production-us-east-1
  - cluster:
      certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURHakNDQWdLZ0F3SUJBZ0lKQVBIOFhpa05Xb2hSTUEwR0NTcUdTSWIzRFFFQkJRVUFNQkl4RURBT0JnTlYKQkFNVEIydDFZbVV0WTJFd0hoY05NVFl3TXpJeE1qQXpNakEzV2hjTk5ETXdPREEzTWpBek1qQTNXakFTTVJBdwpEZ1lEVlFRREV3ZHJkV0psTFdOaE1JSUJJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBCnM3cmVrR29OUE13VEttMkdtdHFmMnhILzUzNS9FT3pEa0gxOStsWnRRZytsQlZuN3RteldwQlZ4NEtmQVZwb1gKcTRRQldiNUNkeDdSUnJkV0ViVEFyVjRpZG1EMmRqU0krWk01TEY0UmVxalhQbURFWWxLRDJQcTVsajR1K0FBMgpYS1VVRXBVZEVOR3cwbXFpSGxDS1Y5dnhUdGlGcmtkSXZ2UXI2QkgyNk5UNk1qY25ZT2RhTVBodFcvOU8zc29tCk5VaEFNVmZ2OC9qaHhaL0trTTlLb0VRM21GbnIzLzh5QzRmbjhjZUZuQ1NjenA5eE96MU1DZjgvOVhFR0hTL0IKZXhZblRQckdTY0R1Z0lQZFBGK1RidzFkbEF2K2QyeFppd1Q3VVQ2QjdjcVovZmZ5enRHeFF6TDhFSy9mYWdGMApzNGF2RlMvU3hwMVNPamhjTGJSNWhRSURBUUFCbzNNd2NUQWRCZ05WSFE0RUZnUVVtSFpZd21oMFlhZU4yK0V1CkliQmVDbUgxUk13d1FnWURWUjBqQkRzd09ZQVVtSFpZd21oMFlhZU4yK0V1SWJCZUNtSDFSTXloRnFRVU1CSXgKRURBT0JnTlZCQU1UQjJ0MVltVXRZMkdDQ1FEeC9GNHBEVnFJVVRBTUJnTlZIUk1FQlRBREFRSC9NQTBHQ1NxRwpTSWIzRFFFQkJRVUFBNElCQVFDRGp6WXdMOXBBU2xSUHRPNU0xeEREUDF0MzdmR3NKdWU0RGR0TnlJS0gvS0VaCnVIKzlaQnRSekoyZ3JtRHVQK1ArWjRrTHk0S2xPYm93R2VvU0VpVEhkU2RDMFdjeE1nbGhpa1RlQTEzMTZVZHgKREVua0x1YTQwbmJpUDJvTUdVbmkzU3VkWWNkbVlWa3J1NWFyYVV2elRCOTJTOHJ6amxXblZDZFArVE43SXdIdgp4emtPQUNvWlViQlBPaDJDd2hicWhZTnRkd09VL3hXQmw0bkJJL2dzTy9sOWQyRVpyN2pEdVlZcmpUNXFpa2FvCnZHVFpvS1FZMmgwYnRaeVlOTWxXNVI0YkFRZmJjNzArTWpDYjFPeGhRUVdXcTQ3aTZkYWt5UjRSaU1oZUJ3QVkKbDhwdEhLZnJWVnA2TUFmWDFQa1R2UXJrNmxPakYrbDdSdVNOMk1lcQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
      server: https://k8s-hurley-master-us-west-2.beta.hurley.hbo.com
    name: hurley-beta-us-west-2
  - cluster:
      certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tDQpNSUlER2pDQ0FnS2dBd0lCQWdJSkFNUU8vWXZuanBaZU1BMEdDU3FHU0liM0RRRUJCUVVBTUJJeEVEQU9CZ05WDQpCQU1UQjJ0MVltVXRZMkV3SGhjTk1UWXdNekl6TVRReU5qTTNXaGNOTkRNd09EQTVNVFF5TmpNM1dqQVNNUkF3DQpEZ1lEVlFRREV3ZHJkV0psTFdOaE1JSUJJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBDQo1N3VNenIraXVyeHZ4L2NQWWZtbXJCWFJMWmMvUDBiU1llUFlRR1RHQnpTM05hdTFOQWk5RXRoTm9TNUw4Kzg4DQpXYjFiOHhRMXVZYlROWENSbS9GYU4xbks1bmxwbDhUV2lGb2dQZnBtU2FOM2RaQTNTRmVJUjVldFEwc0s5RjgrDQo1M0l0T2VmeEYvU3dCNFlWVzFrb0VvN1dWcXQ2WVFBdVNyOWw0cHFXS0tSMjJPWmtJVjdnRUFqWXFTNWpObUlvDQpESlQ1ZWdWMnpSTFNWSGlTUURUb0UxRGlpTGo5VDhveSs2WDlJVnoxMUhiTkZtODBuK3piM1NtUzlFbnNoZkZtDQpKVzVOdFFUdjk2V2F3VE03c2RLakhoOGdkUW1ZbjBSM210ZEJHSzhJZmVXQWpBTk1WYkx0L1JvTjlyL3Q2YW1sDQpIZlY2UVFORmNtSi9xNk10dDkvdVF3SURBUUFCbzNNd2NUQWRCZ05WSFE0RUZnUVVKYkovSDdlQTdBQ0l4M0xUDQorM1dFQitxOFRPTXdRZ1lEVlIwakJEc3dPWUFVSmJKL0g3ZUE3QUNJeDNMVCszV0VCK3E4VE9PaEZxUVVNQkl4DQpFREFPQmdOVkJBTVRCMnQxWW1VdFkyR0NDUURFRHYyTDU0NldYakFNQmdOVkhSTUVCVEFEQVFIL01BMEdDU3FHDQpTSWIzRFFFQkJRVUFBNElCQVFBMFdRcFZMWllWcVBWdllxM0xFQ25TUjBUcFovbW9sMGZQK0ZKVmRjMGM4bVZiDQpMa213bFFmWURXeW5VckFsTnVSa0pScWtyVWJuWnJ1YXRVNG0xMmZBTVRuY2VnMWhhYmt1c3N2d2M1ZEFDeUNlDQpIanVHSGhSTzhNOHJDMjAyRStMdlpsSi9udUtUTERVMUxoamFlOEEzWUNRYjEwVjU0WnhhUlRod0VTZ2FybmVKDQphdmEzdWs5VVd2SmF1MW1ydllTZnF6TGcwZnAwdDVIdzVQcHp5aUw3WHFQOUZOY3ZRb3FaNkVGcmFneFlHYVNMDQpiQ0kxVnlRWThKNndwUWdJV0JMQ21hZW5pNStLMmRrT3ZtcVNLUVVMOGR0RExlQUN2d1lCUjBXa2JtV0VUZ0lRDQpEL3dPbVV2OU5BYmQ1bWkvZllqenJXMW5QVjZWRjFCdXpyTHRtZCs0DQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0t
      server: https://k8s-hurley-master-us-east-1.beta.hurley.hbo.com
    name: hurley-beta-us-east-1
  - cluster:
      certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURHakNDQWdLZ0F3SUJBZ0lKQVBIOFhpa05Xb2hSTUEwR0NTcUdTSWIzRFFFQkJRVUFNQkl4RURBT0JnTlYKQkFNVEIydDFZbVV0WTJFd0hoY05NVFl3TXpJeE1qQXpNakEzV2hjTk5ETXdPREEzTWpBek1qQTNXakFTTVJBdwpEZ1lEVlFRREV3ZHJkV0psTFdOaE1JSUJJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBCnM3cmVrR29OUE13VEttMkdtdHFmMnhILzUzNS9FT3pEa0gxOStsWnRRZytsQlZuN3RteldwQlZ4NEtmQVZwb1gKcTRRQldiNUNkeDdSUnJkV0ViVEFyVjRpZG1EMmRqU0krWk01TEY0UmVxalhQbURFWWxLRDJQcTVsajR1K0FBMgpYS1VVRXBVZEVOR3cwbXFpSGxDS1Y5dnhUdGlGcmtkSXZ2UXI2QkgyNk5UNk1qY25ZT2RhTVBodFcvOU8zc29tCk5VaEFNVmZ2OC9qaHhaL0trTTlLb0VRM21GbnIzLzh5QzRmbjhjZUZuQ1NjenA5eE96MU1DZjgvOVhFR0hTL0IKZXhZblRQckdTY0R1Z0lQZFBGK1RidzFkbEF2K2QyeFppd1Q3VVQ2QjdjcVovZmZ5enRHeFF6TDhFSy9mYWdGMApzNGF2RlMvU3hwMVNPamhjTGJSNWhRSURBUUFCbzNNd2NUQWRCZ05WSFE0RUZnUVVtSFpZd21oMFlhZU4yK0V1CkliQmVDbUgxUk13d1FnWURWUjBqQkRzd09ZQVVtSFpZd21oMFlhZU4yK0V1SWJCZUNtSDFSTXloRnFRVU1CSXgKRURBT0JnTlZCQU1UQjJ0MVltVXRZMkdDQ1FEeC9GNHBEVnFJVVRBTUJnTlZIUk1FQlRBREFRSC9NQTBHQ1NxRwpTSWIzRFFFQkJRVUFBNElCQVFDRGp6WXdMOXBBU2xSUHRPNU0xeEREUDF0MzdmR3NKdWU0RGR0TnlJS0gvS0VaCnVIKzlaQnRSekoyZ3JtRHVQK1ArWjRrTHk0S2xPYm93R2VvU0VpVEhkU2RDMFdjeE1nbGhpa1RlQTEzMTZVZHgKREVua0x1YTQwbmJpUDJvTUdVbmkzU3VkWWNkbVlWa3J1NWFyYVV2elRCOTJTOHJ6amxXblZDZFArVE43SXdIdgp4emtPQUNvWlViQlBPaDJDd2hicWhZTnRkd09VL3hXQmw0bkJJL2dzTy9sOWQyRVpyN2pEdVlZcmpUNXFpa2FvCnZHVFpvS1FZMmgwYnRaeVlOTWxXNVI0YkFRZmJjNzArTWpDYjFPeGhRUVdXcTQ3aTZkYWt5UjRSaU1oZUJ3QVkKbDhwdEhLZnJWVnA2TUFmWDFQa1R2UXJrNmxPakYrbDdSdVNOMk1lcQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
      server: https://k8s-hurley-master-us-west-2.nonprod.hurley.hbo.com
    name: hurley-nonprod-us-west-2
  - cluster:
      certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURHakNDQWdLZ0F3SUJBZ0lKQVBIOFhpa05Xb2hSTUEwR0NTcUdTSWIzRFFFQkJRVUFNQkl4RURBT0JnTlYKQkFNVEIydDFZbVV0WTJFd0hoY05NVFl3TXpJeE1qQXpNakEzV2hjTk5ETXdPREEzTWpBek1qQTNXakFTTVJBdwpEZ1lEVlFRREV3ZHJkV0psTFdOaE1JSUJJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBCnM3cmVrR29OUE13VEttMkdtdHFmMnhILzUzNS9FT3pEa0gxOStsWnRRZytsQlZuN3RteldwQlZ4NEtmQVZwb1gKcTRRQldiNUNkeDdSUnJkV0ViVEFyVjRpZG1EMmRqU0krWk01TEY0UmVxalhQbURFWWxLRDJQcTVsajR1K0FBMgpYS1VVRXBVZEVOR3cwbXFpSGxDS1Y5dnhUdGlGcmtkSXZ2UXI2QkgyNk5UNk1qY25ZT2RhTVBodFcvOU8zc29tCk5VaEFNVmZ2OC9qaHhaL0trTTlLb0VRM21GbnIzLzh5QzRmbjhjZUZuQ1NjenA5eE96MU1DZjgvOVhFR0hTL0IKZXhZblRQckdTY0R1Z0lQZFBGK1RidzFkbEF2K2QyeFppd1Q3VVQ2QjdjcVovZmZ5enRHeFF6TDhFSy9mYWdGMApzNGF2RlMvU3hwMVNPamhjTGJSNWhRSURBUUFCbzNNd2NUQWRCZ05WSFE0RUZnUVVtSFpZd21oMFlhZU4yK0V1CkliQmVDbUgxUk13d1FnWURWUjBqQkRzd09ZQVVtSFpZd21oMFlhZU4yK0V1SWJCZUNtSDFSTXloRnFRVU1CSXgKRURBT0JnTlZCQU1UQjJ0MVltVXRZMkdDQ1FEeC9GNHBEVnFJVVRBTUJnTlZIUk1FQlRBREFRSC9NQTBHQ1NxRwpTSWIzRFFFQkJRVUFBNElCQVFDRGp6WXdMOXBBU2xSUHRPNU0xeEREUDF0MzdmR3NKdWU0RGR0TnlJS0gvS0VaCnVIKzlaQnRSekoyZ3JtRHVQK1ArWjRrTHk0S2xPYm93R2VvU0VpVEhkU2RDMFdjeE1nbGhpa1RlQTEzMTZVZHgKREVua0x1YTQwbmJpUDJvTUdVbmkzU3VkWWNkbVlWa3J1NWFyYVV2elRCOTJTOHJ6amxXblZDZFArVE43SXdIdgp4emtPQUNvWlViQlBPaDJDd2hicWhZTnRkd09VL3hXQmw0bkJJL2dzTy9sOWQyRVpyN2pEdVlZcmpUNXFpa2FvCnZHVFpvS1FZMmgwYnRaeVlOTWxXNVI0YkFRZmJjNzArTWpDYjFPeGhRUVdXcTQ3aTZkYWt5UjRSaU1oZUJ3QVkKbDhwdEhLZnJWVnA2TUFmWDFQa1R2UXJrNmxPakYrbDdSdVNOMk1lcQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
      server: https://k8s-hurley-master-us-east-1.nonprod.hurley.hbo.com
    name: hurley-nonprod-us-east-1
  - cluster:
      certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tDQpNSUlER2pDQ0FnS2dBd0lCQWdJSkFQSDhYaWtOV29oUk1BMEdDU3FHU0liM0RRRUJCUVVBTUJJeEVEQU9CZ05WDQpCQU1UQjJ0MVltVXRZMkV3SGhjTk1UWXdNekl4TWpBek1qQTNXaGNOTkRNd09EQTNNakF6TWpBM1dqQVNNUkF3DQpEZ1lEVlFRREV3ZHJkV0psTFdOaE1JSUJJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBDQpzN3Jla0dvTlBNd1RLbTJHbXRxZjJ4SC81MzUvRU96RGtIMTkrbFp0UWcrbEJWbjd0bXpXcEJWeDRLZkFWcG9YDQpxNFFCV2I1Q2R4N1JScmRXRWJUQXJWNGlkbUQyZGpTSStaTTVMRjRSZXFqWFBtREVZbEtEMlBxNWxqNHUrQUEyDQpYS1VVRXBVZEVOR3cwbXFpSGxDS1Y5dnhUdGlGcmtkSXZ2UXI2QkgyNk5UNk1qY25ZT2RhTVBodFcvOU8zc29tDQpOVWhBTVZmdjgvamh4Wi9La005S29FUTNtRm5yMy84eUM0Zm44Y2VGbkNTY3pwOXhPejFNQ2Y4LzlYRUdIUy9CDQpleFluVFByR1NjRHVnSVBkUEYrVGJ3MWRsQXYrZDJ4Wml3VDdVVDZCN2NxWi9mZnl6dEd4UXpMOEVLL2ZhZ0YwDQpzNGF2RlMvU3hwMVNPamhjTGJSNWhRSURBUUFCbzNNd2NUQWRCZ05WSFE0RUZnUVVtSFpZd21oMFlhZU4yK0V1DQpJYkJlQ21IMVJNd3dRZ1lEVlIwakJEc3dPWUFVbUhaWXdtaDBZYWVOMitFdUliQmVDbUgxUk15aEZxUVVNQkl4DQpFREFPQmdOVkJBTVRCMnQxWW1VdFkyR0NDUUR4L0Y0cERWcUlVVEFNQmdOVkhSTUVCVEFEQVFIL01BMEdDU3FHDQpTSWIzRFFFQkJRVUFBNElCQVFDRGp6WXdMOXBBU2xSUHRPNU0xeEREUDF0MzdmR3NKdWU0RGR0TnlJS0gvS0VaDQp1SCs5WkJ0UnpKMmdybUR1UCtQK1o0a0x5NEtsT2Jvd0dlb1NFaVRIZFNkQzBXY3hNZ2xoaWtUZUExMzE2VWR4DQpERW5rTHVhNDBuYmlQMm9NR1VuaTNTdWRZY2RtWVZrcnU1YXJhVXZ6VEI5MlM4cnpqbFduVkNkUCtUTjdJd0h2DQp4emtPQUNvWlViQlBPaDJDd2hicWhZTnRkd09VL3hXQmw0bkJJL2dzTy9sOWQyRVpyN2pEdVlZcmpUNXFpa2FvDQp2R1Rab0tRWTJoMGJ0WnlZTk1sVzVSNGJBUWZiYzcwK01qQ2IxT3hoUVFXV3E0N2k2ZGFreVI0UmlNaGVCd0FZDQpsOHB0SEtmclZWcDZNQWZYMVBrVHZRcms2bE9qRitsN1J1U04yTWVxDQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0t
      server: https://k8s-hurley-master-us-west-2.perf.hurley.hbo.com
    name: hurley-perf-us-west-2
  - cluster:
      certificate-authority-data: LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURHakNDQWdLZ0F3SUJBZ0lKQVBIOFhpa05Xb2hSTUEwR0NTcUdTSWIzRFFFQkJRVUFNQkl4RURBT0JnTlYKQkFNVEIydDFZbVV0WTJFd0hoY05NVFl3TXpJeE1qQXpNakEzV2hjTk5ETXdPREEzTWpBek1qQTNXakFTTVJBdwpEZ1lEVlFRREV3ZHJkV0psTFdOaE1JSUJJakFOQmdrcWhraUc5dzBCQVFFRkFBT0NBUThBTUlJQkNnS0NBUUVBCnM3cmVrR29OUE13VEttMkdtdHFmMnhILzUzNS9FT3pEa0gxOStsWnRRZytsQlZuN3RteldwQlZ4NEtmQVZwb1gKcTRRQldiNUNkeDdSUnJkV0ViVEFyVjRpZG1EMmRqU0krWk01TEY0UmVxalhQbURFWWxLRDJQcTVsajR1K0FBMgpYS1VVRXBVZEVOR3cwbXFpSGxDS1Y5dnhUdGlGcmtkSXZ2UXI2QkgyNk5UNk1qY25ZT2RhTVBodFcvOU8zc29tCk5VaEFNVmZ2OC9qaHhaL0trTTlLb0VRM21GbnIzLzh5QzRmbjhjZUZuQ1NjenA5eE96MU1DZjgvOVhFR0hTL0IKZXhZblRQckdTY0R1Z0lQZFBGK1RidzFkbEF2K2QyeFppd1Q3VVQ2QjdjcVovZmZ5enRHeFF6TDhFSy9mYWdGMApzNGF2RlMvU3hwMVNPamhjTGJSNWhRSURBUUFCbzNNd2NUQWRCZ05WSFE0RUZnUVVtSFpZd21oMFlhZU4yK0V1CkliQmVDbUgxUk13d1FnWURWUjBqQkRzd09ZQVVtSFpZd21oMFlhZU4yK0V1SWJCZUNtSDFSTXloRnFRVU1CSXgKRURBT0JnTlZCQU1UQjJ0MVltVXRZMkdDQ1FEeC9GNHBEVnFJVVRBTUJnTlZIUk1FQlRBREFRSC9NQTBHQ1NxRwpTSWIzRFFFQkJRVUFBNElCQVFDRGp6WXdMOXBBU2xSUHRPNU0xeEREUDF0MzdmR3NKdWU0RGR0TnlJS0gvS0VaCnVIKzlaQnRSekoyZ3JtRHVQK1ArWjRrTHk0S2xPYm93R2VvU0VpVEhkU2RDMFdjeE1nbGhpa1RlQTEzMTZVZHgKREVua0x1YTQwbmJpUDJvTUdVbmkzU3VkWWNkbVlWa3J1NWFyYVV2elRCOTJTOHJ6amxXblZDZFArVE43SXdIdgp4emtPQUNvWlViQlBPaDJDd2hicWhZTnRkd09VL3hXQmw0bkJJL2dzTy9sOWQyRVpyN2pEdVlZcmpUNXFpa2FvCnZHVFpvS1FZMmgwYnRaeVlOTWxXNVI0YkFRZmJjNzArTWpDYjFPeGhRUVdXcTQ3aTZkYWt5UjRSaU1oZUJ3QVkKbDhwdEhLZnJWVnA2TUFmWDFQa1R2UXJrNmxPakYrbDdSdVNOMk1lcQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==
      server: https://k8s-jenkins-master-us-west-2.api.hbo.com
    name: jenkins-production-us-west-2
contexts:
- context:
    cluster: hurley-production-us-west-2
    user: github-oauth
  name: hurley-production-us-west-2
- context:
    cluster: hurley-production-us-east-1
    user: github-oauth
  name: hurley-production-us-east-1
- context:
    cluster: hurley-beta-us-west-2
    user: github-oauth
  name: hurley-beta-us-west-2
- context:
    cluster: hurley-beta-us-east-1
    user: github-oauth
  name: hurley-beta-us-east-1
- context:
    cluster: hurley-nonprod-us-west-2
    user: github-oauth
  name: hurley-nonprod-us-west-2
- context:
    cluster: hurley-nonprod-us-east-1
    user: github-oauth
  name: hurley-nonprod-us-east-1
- context:
    cluster: hurley-perf-us-west-2
    user: github-oauth
  name: hurley-perf-us-west-2
- context:
    cluster: jenkins-production-us-west-2
    user: github-oauth
  name: jenkins-production-us-west-2
</pre>
          </div>
        </div>
        <div class="panel panel-primary">
          <div class="panel-heading">ID Token</div>
          <div class="panel-body">
            <pre class="pre-y-scrollable">{{ .IDToken }}</pre>
          </div>
        </div>
        <div class="panel panel-primary">
          <div class="panel-heading">Refresh Token</div>
          <div class="panel-body">
            {{ if .RefreshToken }}
              <pre>{{ .RefreshToken }}</pre>
              <form action="{{ .RedirectURL }}" method="post">
                  <input type="hidden" name="refresh_token" value="{{ .RefreshToken }}">
                  <button type="submit" class="btn btn-success">
                  <span class="fa fa-refresh"></span>
                  Redeem New Tokens
                </button>
              </form>
            {{ end }}
          </div>
        </div>
        <div class="panel panel-info">
          <div class="panel-heading">Claims - Information Only</div>
          <div class="panel-body">
            <pre>{{ .Claims }}</pre>
          </div>
        </div>
      </div>
    </div>
  </body>
</html>`))

func renderToken(w http.ResponseWriter, redirectURL, idToken, refreshToken string, claims []byte) {
    renderTemplate(w, tokenTmpl, tokenTmplData{
        IDToken:      idToken,
        RefreshToken: refreshToken,
        RedirectURL:  redirectURL,
        Claims:       string(claims),
    })
}

func renderTemplate(w http.ResponseWriter, tmpl *template.Template, data interface{}) {
    err := tmpl.Execute(w, data)
    if err == nil {
        return
    }

    switch err := err.(type) {
    case *template.Error:
        // An ExecError guarantees that Execute has not written to the underlying reader.
        log.Printf("Error rendering template %s: %s", tmpl.Name(), err)

        // TODO(ericchiang): replace with better internal server error.
        http.Error(w, "Internal server error", http.StatusInternalServerError)
    default:
        // An error with the underlying write, such as the connection being
        // dropped. Ignore for now.
    }
}
