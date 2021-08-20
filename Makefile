build:
	go build -ldflags="-w -s"

install:
	cp proxy-check /usr/bin/proxy-check

all: build install

clean:
	rm -f proxy-check /usr/bin/proxy-check
