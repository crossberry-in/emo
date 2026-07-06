// Package runtime defines the wire protocol between the emo dev server and the
// emo Go preview app running on Android.
//
// All messages are JSON-encoded and sent over a single WebSocket connection.
// The protocol is intentionally minimal: the server pushes vtree snapshots and
// the client emits user events back.
package runtime

import "time"

// MessageKind enumerates message types exchanged over the WebSocket.
type MessageKind string

const (
	// Server → Client
	KindHello    MessageKind = "hello"     // initial handshake
	KindVTree    MessageKind = "vtree"     // full vtree snapshot
	KindPatch    MessageKind = "patch"     // incremental patch (reserved for v2)
	KindReload   MessageKind = "reload"    // ask client to fully reload activity
	KindToast    MessageKind = "toast"     // show a transient toast
	KindLog      MessageKind = "log"       // server-side log line for on-device console
	KindPlugin   MessageKind = "plugin"    // plugin invocation result
	KindError    MessageKind = "error"     // server-side error report

	// Client → Server
	KindEvent    MessageKind = "event"     // user event (click, change, ...)
	KindHandshake MessageKind = "handshake" // client announces itself
	KindAck      MessageKind = "ack"       // acknowledge a server message
)

// Message is the envelope for every WebSocket frame.
type Message struct {
	Kind     MessageKind `json:"kind"`
	Seq      int64       `json:"seq,omitempty"`
	TS       time.Time   `json:"ts,omitempty"`
	Payload  any         `json:"payload,omitempty"`
}

// HelloPayload is sent by the server immediately after the WebSocket opens.
type HelloPayload struct {
	ServerName string `json:"serverName"`
	Version    string `json:"version"`
	ProjectID  string `json:"projectId"`
}

// HandshakePayload is sent by the client to announce itself.
type HandshakePayload struct {
	Client   string `json:"client"`   // "emo-go-android"
	Device   string `json:"device"`   // emulator serial or model name
	Android  string `json:"android"`  // Android API level
	AppVer   string `json:"appVer"`   // emo Go preview app version
}

// VTreePayload carries a full vtree snapshot plus a manifest of handler tokens
// the client should report back when events fire.
type VTreePayload struct {
	Root    any    `json:"root"`    // dsl.Element tree
	Hash    string `json:"hash"`    // short hash of the tree (for dedup)
	Reason  string `json:"reason"`  // "init" | "state" | "file" | "hotswap"
}

// EventPayload is what the client sends when a user interacts with the UI.
type EventPayload struct {
	Token   string `json:"token"`   // matches dsl.HandlerRef.Token
	Event   string `json:"event"`   // "click" | "change" | ...
	Value   any    `json:"value,omitempty"` // e.g. new text for onChange
	Element string `json:"element,omitempty"` // element ID that fired the event
}

// ToastPayload asks the client to show a transient message.
type ToastPayload struct {
	Text     string `json:"text"`
	Duration string `json:"duration,omitempty"` // "short" | "long"
}

// LogPayload forwards a server-side log line to the on-device console.
type LogPayload struct {
	Level string `json:"level"` // "debug" | "info" | "warn" | "error"
	Msg   string `json:"msg"`
}

// ErrorPayload reports a server-side compilation or runtime error so the
// client can render an error overlay instead of a stale tree.
type ErrorPayload struct {
	Title   string `json:"title"`
	Message string `json:"message"`
	Stack   string `json:"stack,omitempty"`
	File    string `json:"file,omitempty"`
	Line    int    `json:"line,omitempty"`
}

// PluginPayload carries a plugin invocation result (camera photo, GPS fix, ...).
type PluginPayload struct {
	Plugin string `json:"plugin"` // "camera" | "location" | "storage"
	Method string `json:"method"` // "takePhoto" | "getCurrentPosition" | ...
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}
