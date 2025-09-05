package template

type IModel interface {
	GetModelID() string
	GetHTMLTemplate() string
	GetCSSTemplate() string
	GetJSTemplate() string
	GetParam() map[string]interface{}
}

type IComponent interface {
	GetID() string
	GetNo() int
	GetParentID() string
	GetModelID() string
	GetModel() IModel
	GetHTML() string
	GetCSS() string
	GetJS() string
	GetParam() map[string]string
	GetChildren() []IComponent

	SetID(string)
	SetModel(IModel)
	SetHTML(string)
	SetCSS(string)
	SetJS(string)
	SetParam(map[string]string)
	SetChildren([]IComponent)
}

type BaseModel struct {
	ID           string                 `json:"ID"`
	HTMLTemplate string                 `json:"HTMLTemplate"`
	CSSTemplate  string                 `json:"CSSTemplate"`
	JSTemplate   string                 `json:"JSTemplate"`
	Param        map[string]interface{} `json:"param"`
}

func (m *BaseModel) GetModelID() string               { return m.ID }
func (m *BaseModel) GetHTMLTemplate() string          { return m.HTMLTemplate }
func (m *BaseModel) GetCSSTemplate() string           { return m.CSSTemplate }
func (m *BaseModel) GetJSTemplate() string            { return m.JSTemplate }
func (m *BaseModel) GetParam() map[string]interface{} { return m.Param }

type BaseComponent struct {
	ComponentID       string            `json:"componentID"`
	No                int               `json:"no"`
	ModelID           string            `json:"modelID"`
	Model             IModel            `json:"-"`
	ModelRelationID   int               `json:"modelRelationID"`
	ParentComponentID string            `json:"parentComponentID"`
	Param             map[string]string `json:"param"`
	HTML              string            `json:"HTML"`
	CSS               string            `json:"CSS"`
	JS                string            `json:"JS"`
	Children          []IComponent      `json:"-"`
}

func (c *BaseComponent) GetID() string               { return c.ComponentID }
func (c *BaseComponent) GetNo() int                  { return c.No }
func (c *BaseComponent) GetParentID() string         { return c.ParentComponentID }
func (c *BaseComponent) GetModelID() string          { return c.ModelID }
func (c *BaseComponent) GetModel() IModel            { return c.Model }
func (c *BaseComponent) GetParam() map[string]string { return c.Param }
func (c *BaseComponent) GetHTML() string             { return c.HTML }
func (c *BaseComponent) GetCSS() string              { return c.CSS }
func (c *BaseComponent) GetJS() string               { return c.JS }
func (c *BaseComponent) GetChildren() []IComponent   { return c.Children }

func (c *BaseComponent) SetID(ID string)                   { c.ComponentID = ID }
func (c *BaseComponent) SetNo(no int)                      { c.No = no }
func (c *BaseComponent) SetParentID(parenID string)        { c.ParentComponentID = parenID }
func (c *BaseComponent) SetModel(model IModel)             { c.Model = model }
func (c *BaseComponent) SetParam(param map[string]string)  { c.Param = param }
func (c *BaseComponent) SetHTML(html string)               { c.HTML = html }
func (c *BaseComponent) SetCSS(css string)                 { c.CSS = css }
func (c *BaseComponent) SetJS(js string)                   { c.JS = js }
func (c *BaseComponent) SetChildren(children []IComponent) { c.Children = children }
