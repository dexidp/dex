# Adopters

This is a list of production adopters of Dex (in alphabetical order):

- [Banzai Cloud](https://banzaicloud.com) is using Dex for authenticating to its Pipeline control plane and also to authenticate users against provisioned Kubernetes clusters (via Kubernetes OIDC support).
- [Chef](https://chef.io) uses Dex for authenticating users in [Chef Automate](https://automate.chef.io/). The code is Open Source, available at [`github.com/chef/automate`](https://github.com/chef/automate).
- [JuliaBox](https://juliabox.com/) is leveraging federated OIDC provided by Dex for authenticating users to their compute infrastructure based on Kubernetes.
- [Kyma](https://kyma-project.io) is using Dex to authenticate access to Kubernetes API server (even for managed Kubernetes like Google Kubernetes Engine or Azure Kubernetes Service) and for protecting web UI of [Kyma Console](https://github.com/kyma-project/console) and other UIs integrated in Kyma ([Grafana](https://github.com/grafana/grafana), [Loki](https://github.com/grafana/loki), and [Jaeger](https://github.com/jaegertracing/jaeger)). Kyma is an open-source project ([`github.com/kyma-project`](https://github.com/kyma-project/kyma)) designed natively on Kubernetes, that allows you to extend and customize your applications in a quick and modern way, using serverless computing or microservice architecture. 
- [Pusher](https://pusher.com) uses Dex for authenticating users across their Kubernetes infrastructure (using Kubernetes OIDC support) in conjunction with the [OAuth2 Proxy](https://github.com/pusher/oauth2_proxy) for protecting web UIs.
- [Pydio](https://pydio.com/) Pydio Cells is an open source sync & share platform written in Go. Cells is using Dex as an OIDC service for authentication and authorizations. Check out [Pydio Cells repository](https://github.com/pydio/cells) for more information and/or to contribute.
