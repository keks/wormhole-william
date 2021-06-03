package codes

type Code int

const (
	OK = Code(iota)
	ERR_NO_CLIENT
	ERR_SEND_TEXT
	ERR_SEND_TEXT_RESULT
	ERR_RECV_TEXT
)

func (c Code) String() string {
	switch c {
	case OK:
		return "ok"
	case ERR_NO_CLIENT:
		return "client instance missing"
	case ERR_SEND_TEXT:
		return "error starting text send"
	case ERR_SEND_TEXT_RESULT:
		return "error during text send"
	case ERR_RECV_TEXT:
		return "error during text receive"
	}
	return "unknown error"
}
