```bash
root@ubuntu-mass:~# helm repo list
NAME      	URL
jetstack  	https://charts.jetstack.io
istio     	https://istio-release.storage.googleapis.com/charts
nvidia    	https://helm.ngc.nvidia.com/nvidia
higress.io	https://higress.cn/helm-charts
root@ubuntu-mass:~#

root@ubuntu-mass:~# helm list -A
NAME                	NAMESPACE          	REVISION	UPDATED                                	STATUS  	CHART                      	APP VERSION
cert-manager        	cert-manager       	1       	2026-06-15 10:53:12.905068365 +0000 UTC	deployed	cert-manager-v1.17.0       	v1.17.0
gpu-operator        	gpu-operator       	1       	2026-06-22 04:30:11.572156018 +0000 UTC	deployed	gpu-operator-v26.3.2       	v26.3.2
higress             	higress-system     	1       	2026-06-16 05:33:04.011920161 +0000 UTC	deployed	higress-2.2.2              	2.2.2
istio-base          	istio-system       	1       	2026-06-15 10:54:42.767280748 +0000 UTC	deployed	base-1.27.1                	1.27.1
istio-ingressgateway	istio-system       	1       	2026-06-15 10:56:25.835804594 +0000 UTC	deployed	gateway-1.27.1             	1.27.1
istiod              	istio-system       	1       	2026-06-15 10:54:49.039054969 +0000 UTC	deployed	istiod-1.27.1              	1.27.1
knative-operator    	knative-operator   	1       	2026-06-15 10:59:38.770790158 +0000 UTC	deployed	knative-operator-1.21.1
monitor             	onecloud-monitoring	1       	2026-06-15 10:01:28.237094225 +0000 UTC	deployed	monitor-stack-v2-55.11.0   	v0.70.0
monitor-minio       	onecloud-monitoring	1       	2026-06-15 10:00:48.584279931 +0000 UTC	deployed	minio-8.0.6                	master
traefik             	kube-system        	2       	2026-06-15 10:45:55.207304247 +0000 UTC	deployed	traefik-25.0.2+up25.0.0    	v2.10.5
traefik-crd         	kube-system        	2       	2026-06-15 10:45:55.159170748 +0000 UTC	deployed	traefik-crd-25.0.2+up25.0.0	v2.10.5
root@ubuntu-mass:~#

helm get values cert-manager --namespace cert-manager
helm get values gpu-operator --namespace gpu-operator
helm get values higress --namespace higress-system
helm get values istio-base --namespace istio-system
helm get values istio-ingressgateway --namespace istio-system
helm get values istiod --namespace istio-system
helm get values knative-operator --namespace knative-operator
helm get values monitor --namespace onecloud-monitoring
helm get values monitor-minio --namespace onecloud-monitoring
helm get values traefik --namespace kube-system
helm get values traefik-crd --namespace kube-system

helm get all cert-manager --namespace cert-manager > ./release/cert-manager.yaml 2>&1
helm get all gpu-operator --namespace gpu-operator > ./release/gpu-operator.yaml 2>&1
helm get all higress --namespace higress-system > ./release/higress.yaml 2>&1
helm get all istio-base --namespace istio-system > ./release/istio-base.yaml 2>&1
helm get all istio-ingressgateway --namespace istio-system > ./release/istio-ingressgateway.yaml 2>&1
helm get all istiod --namespace istio-system > ./release/istiod.yaml 2>&1
helm get all knative-operator --namespace knative-operator > ./release/knative-operator.yaml 2>&1
helm get all monitor --namespace onecloud-monitoring > ./release/monitor.yaml 2>&1
helm get all monitor-minio --namespace onecloud-monitoring > ./release/monitor-minio.yaml 2>&1
helm get all traefik --namespace kube-system > ./release/traefik.yaml 2>&1
helm get all traefik-crd --namespace kube-system > ./release/traefik-crd.yaml 2>&1


```