build:
	go build -ldflags="-w -s"

install:
	cp proxy-check /usr/bin/proxy-check

termux-install:
	mv proxy-check  /data/data/com.termux/files/usr/bin/proxy-check

all: build install

termux-all: build termux-install

clean:
	rm -f proxy-check /usr/bin/proxy-check

termux-clean:
	rm -f proxy-check /data/data/com.termux/files/usr/bin/proxy-check
