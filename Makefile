ppdf:
	go tool pprof --pdf profile.dat > profile.pdf
	evince profile.pdf
