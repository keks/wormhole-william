package codes

type Code int

const (
	OK = Code(iota)
	ERR_NO_CLIENT
)

func (c Code) String() string {
	switch c {
	case OK:
		return "ok"
	case ERR_NO_CLIENT:
		return "client instance missing"
	}
	return "unknown error"
}
