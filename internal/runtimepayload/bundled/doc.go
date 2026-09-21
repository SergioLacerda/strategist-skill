// Package bundled registers the runtime payload compiled into a
// -tags strategist_payload build. It is a separate package on purpose: the tagged
// files import internal/embed, and keeping them inside runtimepayload created an
// import cycle in the tests of internal/embed (embed's tests import install,
// which imports runtimepayload) that only showed up when compiling with the tag.
// Only cmd/strategist imports this package, and only under the tag.
package bundled
