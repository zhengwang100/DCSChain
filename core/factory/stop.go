package factory

import "server"

// StopAll: stop all servers
func StopAll(servers []*server.Server) {
	for _, s := range servers {
		s.Orderer.Stop()
		if s.Clients["c_0"].Conn != nil {
			(*s.Clients["c_0"].Conn).Close()
		}
	}
}
