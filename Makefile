all: tor-ingress-controller

push: docker
	docker push kragniz/tor-ingress-controller:latest

docker: tor-ingress-controller
	docker build . -t kragniz/tor-ingress-controller:latest

tor-ingress-controller: Makefile vendor main.go tor/*.go
	CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o tor-ingress-controller

vendor: Gopkg.toml
	dep ensure

