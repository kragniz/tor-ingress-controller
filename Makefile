all:
	go build -i -o tor-ingress-controller

docker:
	docker build . -t kragniz/tor-ingress-controller:latest

push:
	docker push kragniz/tor-ingress-controller:latest
