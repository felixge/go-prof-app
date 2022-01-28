BUILD_VARIANT       ?= release
DD_TRACE_GO_VERSION ?= v1.36.0
BUILD_TAGS          := $(BUILD_VARIANT),$(BUILD_TAGS)

.PHONY: install
install:
	go get -v gopkg.in/DataDog/dd-trace-go.v1@$(DD_TRACE_GO_VERSION)
	go get -v .
	go install -v -ldflags "-X main.version=$(BUILD_VARIANT)/`git describe --tags HEAD`" -tags "$(BUILD_TAGS)"
