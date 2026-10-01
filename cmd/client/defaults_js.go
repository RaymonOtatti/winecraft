//go:build js && wasm

package main

import "syscall/js"

// defaults reads the browser page: the game server is the page's own host,
// and the join form (index.html) leaves the name and code in WINECRAFT_JOIN.
// ?name=…&code=… still work, for quick tests.
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
	st := startup{server: scheme + "//" + loc.Get("host").String() + "/ws", name: get("name"), code: get("code"), walk: get("walk")}
	if j := js.Global().Get("WINECRAFT_JOIN"); j.Truthy() {
		st.name, st.code = j.Get("name").String(), j.Get("code").String()
	}
	return st
}
