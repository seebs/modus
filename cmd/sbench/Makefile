run: sbench
	./sbench

sbench: *.go
	go build -race

lint:
	go fmt
	go vet
	golint

full-lint: lint
	staticcheck

ppdf:
	go tool pprof --pdf profile.dat > profile.pdf
	evince profile.pdf
