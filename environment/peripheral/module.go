package peripheral

const (
	MODULE_PEREPHERAL = "peripheral"

	ACTION_ADMIN_VIEW = "admin_view"
	ACTION_ADMIN_EDIT = "admin_edit"

	ACTION_VIEW = "view"
	ACTION_EDIT = "edit"
)

var AdminActions = map[string]string{
	ACTION_VIEW: ACTION_ADMIN_VIEW,
	ACTION_EDIT: ACTION_ADMIN_EDIT,
}

const (
	DEVICE_SPEAKER            = "speaker"
	DEVICE_EZVIZ_SURVEILLANCE = "ezviz"
)
