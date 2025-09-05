package hjt212

import (
	"errors"
	"log"
	"obsessiontech/common/util"
	"obsessiontech/environment/environment"
)

var REQUEST_NOT_SUPPORT = errors.New("不支持该请求类型")

type Executor interface {
	GetMN() string
	Execute(siteID, QN string, input func() (*Instruction, error), process func(*Instruction), output func(*Instruction) error, close func(error))
}

var routing = make(map[string]func() Executor)

func RegisterExecutor(CN string, fac func() Executor) {
	routing[CN] = fac
}

func (p *HJT212) InvokeExecutor(instruction *Instruction) (Executor, error) {

	log.Printf("匹配请求处理接口 MN[%s] QN[%s] CN[%s]", instruction.MN, instruction.QN, instruction.CN)
	if fac, registered := routing[instruction.CN]; registered {
		exe := fac()
		return exe, nil
	} else {
		m, err := environment.GetModule(p.SiteID)
		if err != nil {
			return nil, err
		}

		for _, proto := range m.Protocols {
			if proto.Protocol == p.GetProtocol() {
				if param, exists := proto.Extra[instruction.CN]; exists {
					if paramMap, ok := param.(map[string]interface{}); ok {
						if executor, exists := paramMap["executor"]; exists {
							if cn, ok := executor.(string); ok {
								if fac, registered := routing[cn]; registered {
									exe := fac()
									if err := util.Clone(param, exe); err != nil {
										return nil, err
									}

									return exe, nil
								}
							}
						}
					}
				}
				log.Println("protocol extension CN not found: ", proto.Extra)
				break
			}
		}

		return nil, REQUEST_NOT_SUPPORT
	}
}
