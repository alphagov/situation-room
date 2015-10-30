VERSION=$(if $(shell git show-ref 2>/dev/null),$(shell git show-ref --head --hash=8|head -n1|cut -f 1 -d ' '))
GIT_DIRTY=$(if $(shell git show-ref 2>/dev/null),$(shell git diff 2> /dev/null | wc -l | tr -d ' '))

BINARY=situation-room

build:
	go build -ldflags "-X main.Version='${VERSION}-${GIT_DIRTY}'"

clean:
	rm ${BINARY}
