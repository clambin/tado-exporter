package tadoproxy

func (proxy *Proxy) GetAllZones() map[int]string {
	response := make(chan map[int]string)
	proxy.AllZones <- response
	return <-response
}

func (proxy *Proxy) GetAllUsers() map[int]string {
	response := make(chan map[int]string)
	proxy.AllUsers <- response
	return <-response
}
