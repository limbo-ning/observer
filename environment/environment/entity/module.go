package entity

const (
	MODULE_ENTITY = "environment_entity"

	ACTION_ENTITY_VIEW   = "view"
	ACTION_ENTITY_EXPORT = "export"
	ACTION_ENTITY_EDIT   = "edit"

	ACTION_ADMIN_VIEW   = "admin_view"
	ACTION_ADMIN_EXPORT = "admin_export"
	ACTION_ADMIN_EDIT   = "admin_edit"
)

var AdminActions = map[string]string{
	ACTION_ENTITY_VIEW:   ACTION_ADMIN_VIEW,
	ACTION_ENTITY_EXPORT: ACTION_ADMIN_EXPORT,
	ACTION_ENTITY_EDIT:   ACTION_ADMIN_EDIT,
}
