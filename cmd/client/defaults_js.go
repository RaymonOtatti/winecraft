//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/RaymonOtatti/winecraft/internal/client"
)

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
		if t := j.Get("token"); t.Truthy() {
			st.token = t.String()
		}
	}
	return st
}

// deviceToken: in the browser the page made the token (localStorage).
func deviceToken() string { return "" }

// prefs keeps settings in localStorage; if storage is blocked (private
// mode), they last for this visit only.
type prefs struct{ mem client.MemPrefs }

func newPrefs() client.Prefs { return prefs{mem: client.MemPrefs{}} }

func (p prefs) Get(k string) string {
	defer func() { recover() }() // a storage access can throw
	if v := js.Global().Get("localStorage").Call("getItem", "winecraft."+k); !v.IsNull() {
		return v.String()
	}
	return p.mem.Get(k)
}

func (p prefs) Set(k, v string) {
	p.mem.Set(k, v)
	defer func() { recover() }()
	js.Global().Get("localStorage").Call("setItem", "winecraft."+k, v)
}
