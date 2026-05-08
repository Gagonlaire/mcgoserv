package nbtpath

type Sink interface {
	Punct(s string)
	Key(s string)
	String(s string)
	Number(s string)
	Type(s string)
}
