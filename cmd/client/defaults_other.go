//go:build !(js && wasm)

package main

// defaults: natively there is no page, so flags say everything.
func defaults() startup { return startup{} }
