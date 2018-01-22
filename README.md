tor-ingress-controller
======================

A poorly written ingress controller to expose kubernetes services as [onion
services](https://www.torproject.org/docs/onion-services) on the tor network.

To use, ingresses must be marked with an ingress class of `"tor"`, for example:

```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  annotations:
    kubernetes.io/ingress.class: "tor"
spec:
  backend:
    serviceName: test-service
    servicePort: 80
```
