package authority

type ActionAuth struct {
	UID         int
	Session     string
	ClientAgent string
	Action      string
	RoleSeries  string
	RoleType    string
	RoleID      int
}

type IAuth interface {
	CheckAuth(string, ActionAuthSet, ...interface{}) error
}

type IAuthList []IAuth

type ActionAuthSet []*ActionAuth

func (set ActionAuthSet) GetUID() int {
	if len(set) > 0 {
		return set[0].UID
	}

	return 0
}
func (set ActionAuthSet) GetSession() string {
	if len(set) > 0 {
		return set[0].Session
	}

	return ""
}
func (set ActionAuthSet) GetClientAgent() string {
	if len(set) > 0 {
		return set[0].ClientAgent
	}

	return ""
}

func (set ActionAuthSet) CheckAction(action ...string) bool {

	toCheck := make(map[string]bool)
	for _, c := range action {
		toCheck[c] = true
	}

	for _, a := range set {
		if toCheck[a.Action] {
			return true
		}
	}

	return false
}
