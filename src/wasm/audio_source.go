package main

import "syscall/js"

type AudioSource interface {
	Set(prop string, value any)
	Get(prop string) js.Value
	Call(m string, args ...any) js.Value
}
