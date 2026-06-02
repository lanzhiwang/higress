# Gateway API CRD

```bash

kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.4.0/experimental-install.yaml

CustomResourceDefinition
  gatewayclasses.gateway.networking.k8s.io
  gateways.gateway.networking.k8s.io
  httproutes.gateway.networking.k8s.io
  grpcroutes.gateway.networking.k8s.io
  tcproutes.gateway.networking.k8s.io
  tlsroutes.gateway.networking.k8s.io
  udproutes.gateway.networking.k8s.io
  backendtlspolicies.gateway.networking.k8s.io
  referencegrants.gateway.networking.k8s.io
  xbackendtrafficpolicies.gateway.networking.x-k8s.io
  xlistenersets.gateway.networking.x-k8s.io
  xmeshes.gateway.networking.x-k8s.io

```

## Envoy Gateway

```bash
CustomResourceDefinition
  backendtlspolicies.gateway.networking.k8s.io
  gatewayclasses.gateway.networking.k8s.io
  gateways.gateway.networking.k8s.io
  grpcroutes.gateway.networking.k8s.io
  httproutes.gateway.networking.k8s.io
  listenersets.gateway.networking.k8s.io
  referencegrants.gateway.networking.k8s.io
  tcproutes.gateway.networking.k8s.io
  tlsroutes.gateway.networking.k8s.io
  udproutes.gateway.networking.k8s.io
  xbackendtrafficpolicies.gateway.networking.x-k8s.io
  xmeshes.gateway.networking.x-k8s.io
  backends.gateway.envoyproxy.io
  backendtrafficpolicies.gateway.envoyproxy.io
  clienttrafficpolicies.gateway.envoyproxy.io
  envoyextensionpolicies.gateway.envoyproxy.io
  envoypatchpolicies.gateway.envoyproxy.io
  envoyproxies.gateway.envoyproxy.io
  httproutefilters.gateway.envoyproxy.io
  securitypolicies.gateway.envoyproxy.io

ValidatingAdmissionPolicy
  safe-upgrades.gateway.networking.k8s.io

ValidatingAdmissionPolicyBinding
  safe-upgrades.gateway.networking.k8s.io

Namespace
  envoy-gateway-system

ServiceAccount
  envoy-gateway
  eg-gateway-helm-certgen

ConfigMap
  envoy-gateway-config

ClusterRole
  eg-gateway-helm-envoy-gateway-role
  eg-gateway-helm-certgen:envoy-gateway-system

ClusterRoleBinding
  eg-gateway-helm-envoy-gateway-rolebinding
  eg-gateway-helm-certgen:envoy-gateway-system

Role
  eg-gateway-helm-infra-manager
  eg-gateway-helm-leader-election-role
  eg-gateway-helm-certgen

RoleBinding
  eg-gateway-helm-infra-manager
  eg-gateway-helm-leader-election-rolebinding
  eg-gateway-helm-certgen

Service
  envoy-gateway

Deployment
  envoy-gateway envoyproxy/gateway:v1.8.0

Job
  eg-gateway-helm-certgen envoyproxy/gateway:v1.8.0

MutatingWebhookConfiguration
  envoy-gateway-topology-injector.envoy-gateway-system

```
