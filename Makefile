code = ssh-authy

all:
	go build ${code}.go
	strip ${code}

multi_arch:
	GOOS=linux GOARC=amd64 go build -ldflags '-w' ${code}.go
	GOOS=darwin GOARC=amd64 go build -ldflags '-w' -o ${code}.osx ${code}.go
	GOOS=windows GOARC=amd64 go build -ldflags '-w' ${code}.go
	strip ${code}
	strip ${code}.osx
	strip ${code}.exe

clean:
	rm -f ${code}

cleanall:
	rm -f ${code}
	rm -f ${code}.osx
	rm -f ${code}.exe
