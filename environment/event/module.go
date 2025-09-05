package event

const (
	MODULE_EVENT = "event"

	ACTION_ADMIN_EDIT = "admin_edit"
	ACTION_ADMIN_VIEW = "admin_view"

	ACTION_EDIT = "edit"
	ACTION_VIEW = "view"
)

var AdminActions = map[string]string{
	ACTION_VIEW: ACTION_ADMIN_VIEW,
	ACTION_EDIT: ACTION_ADMIN_EDIT,
}
