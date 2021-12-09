BUILD_VARIANT=release
DD_TRACE_GO_VERSION:=v1.34.0

.PHONY: install
install:
	go get gopkg.in/DataDog/dd-trace-go.v1@$(DD_TRACE_GO_VERSION)
	go get .
	go install -ldflags "-X main.version=$(BUILD_VARIANT)/`git describe --tags HEAD`" -tags $(BUILD_VARIANT)
