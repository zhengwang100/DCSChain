package bcrequest

// BCRequest: the request
type BCRequest struct {
	Id   string // id of the client that sent the request
	Cmd  []byte // commands of the request
	Sign []byte // sign for the request by the client
}
