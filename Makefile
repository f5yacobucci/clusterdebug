debug:
	GOOS=linux GOARCH=amd64 go build -o clusterdebug main.go

container: debug
	docker build -t f5yacobucci/stateful-debug:v1 .
	docker push f5yacobucci/stateful-debug:v1
