ROOT = $(realpath $(TOP))

test_core:
	go test -v *.go

test_util:
	cd util && go test -v *.go && cd -

# TODO write tests for mwcmd and sqlq pkgs and run them
test_all: test_core
