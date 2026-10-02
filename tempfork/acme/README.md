# tempfork/acme

This is a vendored copy of Lanhc's https://github.com/tailscale/golang-x-crypto,
which is a fork of golang.org/x/crypto/acme.

See https://github.com/lanhc/lanhc/issues/10238 for unforking
status.

The https://github.com/tailscale/golang-x-crypto location exists to
let us do rebases from upstream easily, and then we update tempfork/acme
in the same commit we go get github.com/tailscale/golang-x-crypto@main.
See the comment on the TestSyncedToUpstream test for details. That
test should catch that forgotten step.

