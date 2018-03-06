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

Quickstart
----------

Install into your cluster:

```bash
$ kubectl apply -f hack/example-controller-deployment.yaml
```

Store onion service key
-----------------------

By default, tor-ingress-controller will create a random address for each
ingress you create. If you require a more consistent address, you'll need to
persist the private key.

Save your private key somewhere (this one isn't very private):

```
$ cat super-secret
-----BEGIN RSA PRIVATE KEY-----
MIICXAIBAAKBgQDEV7KcR3ESUI7Gr3TovoRhGSUYfwvCVmC44wKTj/kB/t2bPkmm
uWsV31uRrRBpPPLyZuFu+m8vSztiEbTp5ZpxdctHbglv5ys3EfixrqD9EwxYHkmy
kv6kSnYf5p9p3p8KBFeD02VX5tt3pXuuHlNnvuuWY14tZZKy78KJhkEOEQIDAQAB
AoGAZ0ZZxKovZ5rH/uo7bFEKAKjhQklRPh+BML73k/ae29XbatUQmInfMdoSqEWH
5FMS1z4WRfGkmhPQYH0/0+fZm/bHRDgNADRkMd43XoUiK5q7dn/lheDwJ9tbLvy1
5fN2yBbGQ2n9DsM3CN6DpbGd3N/8rTJrcAfPI51NMR5GegECQQD+S+StqUffN9Hq
l+mh6jTyKKwBAR0+I9GAY1UPBFakRUcY8vRsWM8koSkUHBIgl9llB7dTqEM8l4qj
Vi5EFpCpAkEAxahqihC4+4BUcQm1t8B1zvgZs65evp7A4XVlfRZVC3DJS2mYA+On
eNAM7/sdaFkfvqOM9nTXilxySoQh5surKQJALNTwcfVgKGhM59D0bYk+4FpvSJYL
s8LY0ouwmT8ojzlveWSL1vYpPsny1grE314mA3vCxErr36jP1lABRBu+UQJAU+ti
eIYLE/TzZR7bQU38dshNmUUyQrqCZ/cBBO/jYb0cKeGGQjh41Ul4BLfYT4JvgPBN
nCIVlVAU0mBxSF02qQJBALZsK4cZWWEygXFIcMK6TNlfjP1BGrf/bhjVao0j2sIf
x78TKBDam/6FIZCjH367kkwhyTHfwpeMbMkDrSpug4E=
-----END RSA PRIVATE KEY-----
```

Create a secret from it:

```shell
$ kubectl create secret generic bmy7nlgozpyn26tv --from-file=super-secret
```

Create an ingress with the `tor.kragniz.eu/private-key-secret` annotation, something like:

```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: test-ingress
  annotations:
    kubernetes.io/ingress.class: "tor"
    tor.kragniz.eu/private-key-secret: bmy7nlgozpyn26tv
spec:
  backend:
    serviceName: http-svc
    servicePort: 80
```

Check to make sure the ingress gets the correct onion address for the private key:

```shell
$ kubectl get ingress -o wide
NAME           HOSTS     ADDRESS                  PORTS     AGE
test-ingress   *         bmy7nlgozpyn26tv.onion   80        1h
```
