package protocol

var protocols = make(map[string]func() IProtocol)

func Register(name string, factory func() IProtocol) {
	if _, exists := protocols[name]; exists {
		panic("duplicate protocol:" + name)
	}
	protocols[name] = factory
}

func GetProtocol(protocol string) IProtocol {

	fac, exists := protocols[protocol]
	if !exists {
		return nil
	}

	proto := fac()
	proto.SetProtocol(protocol)

	return proto
}

func GetSupportedProtocols() []string {
	result := make([]string, 0)
	for p := range protocols {
		result = append(result, p)
	}
	return result
}
