install:
	go install -ldflags "-X main.version=`git describe --tags HEAD`"
	
