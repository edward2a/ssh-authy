code = ssh_authy

all:
	go build ${code}.go

multi_arch:
	GOOS=linux GOARC=amd64 go build -ldflags '-w' ${code}.go
	GOOS=darwin GOARC=amd64 go build -ldflags '-w' -o ${code}.osx ${code}.go
	GOOS=windows GOARC=amd64 go build -ldflags '-w' ${code}.go
