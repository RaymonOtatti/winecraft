//go:build js && wasm

package main

import "syscall/js"

// defaults reads the browser page: the game server is the page's own host,
// and ?name=…&code=… fill in the player until the join screen exists.
func defaults() startup {
	loc := js.Global().Get("location")
	scheme := "ws:"
	if loc.Get("protocol").String() == "https:" {
		scheme = "wss:"
	}
	params := js.Global().Get("URLSearchParams").New(loc.Get("search"))
	get := func(k string) string {
		if v := params.Call("get", k); !v.IsNull() {
			return v.String()
		}
		return ""
	}
	return startup{server: scheme + "//" + loc.Get("host").String() + "/ws", name: get("name"), code: get("code")}
}
