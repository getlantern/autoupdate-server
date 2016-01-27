all: clean
	GOOS=linux GOARCH=amd64 go build -o autoupdate-server

clean:
	rm -rf autoupdate-server patches assets

docker:
	docker build -t getlantern/autoudate-server .
