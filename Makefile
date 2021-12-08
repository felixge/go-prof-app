RELEASE_VERSION:=v1.34.0
CANDIDATE_VERSION:=b956dd4cc7bc485f4db19fc3cf3a64fc7c117ca5

GO_INSTALL = go install -ldflags "-X main.version=$(1)/`git describe --tags HEAD`" -tags $1
GO_GET_DD_TRACE_GO=go get gopkg.in/DataDog/dd-trace-go.v1

.PHONY: release
release:
	$(GO_GET_DD_TRACE_GO)@$(RELEASE_VERSION)
	go get .
	$(call GO_INSTALL,release)

.PHONY: candidate
candidate:
	$(GO_GET_DD_TRACE_GO)@$(CANDIDATE_VERSION)
	go get .
	$(call GO_INSTALL,candidate)

.PHONY: test_install
test_install: release candidate
	git checkout go.* # cleanup
