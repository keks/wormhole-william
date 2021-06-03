package codes

type Code int

const (
	ERR_UNKNOWN = Code(iota) - 1
	OK
	ERR_NO_CLIENT
	ERR_SEND_TEXT
	ERR_SEND_TEXT_RESULT
	ERR_RECV_TEXT
	ERR_RECV_TEXT_DATA
)

func (c Code) String() string {
	switch c {
	case ERR_UNKNOWN:
		return "unknown error"
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
	case ERR_RECV_TEXT_DATA:
		return "error during text receive reading data"
	}
	return "unknown error"
}
