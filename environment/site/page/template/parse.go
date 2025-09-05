package template

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

const PLACEHOLDER_PREFIX = "{%"
const PLACEHOLDER_SUBFIX = "%}"

const RESERVED_PARAM = "param"
const RESERVED_ID = "ID"
const RESERVED_PARENT_ID = "parentID"
const RESERVED_CHILDREN = "children"
const RESERVED_CHILDREN_LENGTH = "childrenLength"
const RESERVED_CHILD_INDEX = "childIndex"
const RESERVED_CFRAMEWORK = "cframework"
const RESERVED_TS = "ts"

const BREAK = `
`

func replaceCondition(tpl, key, value string) string {
	if value == "" {
		matcher := regexp.MustCompile(fmt.Sprintf("%s%s\\s(?sU:(.*))\\s%s%s", PLACEHOLDER_PREFIX, key, key, PLACEHOLDER_SUBFIX))
		tpl = matcher.ReplaceAllString(tpl, "")
	}
	conditionMatcher := regexp.MustCompile(fmt.Sprintf("%s%s:(\\S+)\\s", PLACEHOLDER_PREFIX, key))
	matches := conditionMatcher.FindAllStringSubmatch(tpl, -1)
	if matches != nil && len(matches) > 0 {
		for _, sub := range matches {
			matched := false

			if value != "" {
				conditions := strings.Split(sub[1], "|")
				for _, per := range conditions {
					if strings.Contains(value, per) {
						matched = true
						break
					}
				}
			}

			escape := strings.ReplaceAll(sub[1], "|", "\\|")

			if !matched {
				matcher := regexp.MustCompile(fmt.Sprintf("%s%s:%s\\s(?sU:(.*))\\s%s:%s%s", PLACEHOLDER_PREFIX, key, escape, key, escape, PLACEHOLDER_SUBFIX))
				tpl = matcher.ReplaceAllString(tpl, "")
			} else {
				matcher := regexp.MustCompile(fmt.Sprintf("(%s%s:%s\\s)|(\\s%s:%s%s)", PLACEHOLDER_PREFIX, key, escape, key, escape, PLACEHOLDER_SUBFIX))
				tpl = matcher.ReplaceAllString(tpl, "")
			}
		}
	}
	matcher := regexp.MustCompile(fmt.Sprintf("(%s%s\\s)|(\\s%s%s)", PLACEHOLDER_PREFIX, key, key, PLACEHOLDER_SUBFIX))
	return matcher.ReplaceAllString(tpl, "")
}

func replace(tpl, key, value string) string {

	key = fmt.Sprintf("%s%s%s", PLACEHOLDER_PREFIX, key, PLACEHOLDER_SUBFIX)

	if loc := strings.Index(tpl, key); loc == -1 {
		return tpl
	}

	return strings.Replace(tpl, key, value, -1)
}

func fillParam(comp IComponent, components map[string]IComponent, skipJS bool) error {

	if comp.GetParam() == nil {
		return nil
	}

	param := make(map[string]interface{})
	for k := range comp.GetModel().GetParam() {
		param[k] = comp.GetParam()[k]
	}
	param[RESERVED_ID] = comp.GetID()

	parentIDs := make([]string, 0)

	traceParent := comp
	exists := true

	for {
		if traceParent.GetParentID() != "" {
			traceParent, exists = components[traceParent.GetParentID()]
			if !exists {
				break
			}
			parentIDs = append(parentIDs, traceParent.GetID())
			continue
		}
		break
	}

	for i, parentID := range parentIDs {
		key := fmt.Sprintf("%s_%d", RESERVED_PARENT_ID, i)
		param[key] = parentID
	}

	if comp.GetChildren() != nil {
		param[RESERVED_CHILDREN_LENGTH] = len(comp.GetChildren())
	}

	param[RESERVED_CHILD_INDEX] = comp.GetNo()

	param[RESERVED_CFRAMEWORK] = ""

	ts := time.Now().Unix()
	param[RESERVED_TS] = ts

	if !skipJS {
		paramB, err := json.Marshal(param)
		if err != nil {
			return err
		}
		comp.SetJS(replace(comp.GetJS(), RESERVED_PARAM, string(paramB)))
	}

	for k, value := range param {
		var v string
		switch vv := value.(type) {
		case int:
			v = fmt.Sprintf("%d", vv)
		case uint:
			v = fmt.Sprintf("%d", vv)
		case int64:
			v = fmt.Sprintf("%d", vv)
		case string:
			v = vv
		case bool:
			v = fmt.Sprintf("%t", vv)
		default:
			break
		}
		comp.SetHTML(replaceCondition(comp.GetHTML(), k, v))
		comp.SetCSS(replaceCondition(comp.GetCSS(), k, v))
		comp.SetJS(replaceCondition(comp.GetJS(), k, v))

		comp.SetHTML(replace(comp.GetHTML(), k, v))
		comp.SetCSS(replace(comp.GetCSS(), k, v))
		comp.SetJS(replace(comp.GetJS(), k, v))
	}

	return nil
}

func getModelParams(comp IComponent) map[string]byte {

	result := make(map[string]byte)

	for k := range comp.GetModel().GetParam() {
		switch k {
		case RESERVED_CFRAMEWORK:
			break
		case RESERVED_CHILDREN:
		case RESERVED_CHILDREN_LENGTH:
		case RESERVED_CHILD_INDEX:
		case RESERVED_ID:
		case RESERVED_PARAM:
		case RESERVED_PARENT_ID:
		case RESERVED_TS:
		default:
			if _, exists := comp.GetParam()[k]; !exists {
				break
			}
		}
		result[k] = 1
	}
	return result
}

func fillChildren(comp IComponent) error {

	if comp.GetChildren() == nil || len(comp.GetChildren()) == 0 {
		comp.SetHTML(replace(comp.GetHTML(), RESERVED_CHILDREN, ""))
		comp.SetCSS(replace(comp.GetCSS(), RESERVED_CHILDREN, ""))
		comp.SetJS(replace(comp.GetJS(), RESERVED_CHILDREN, ""))
		return nil
	}

	childrenNoMap := make(map[int]IComponent, 0)
	childrenNos := make([]int, 0)

	for _, c := range comp.GetChildren() {

		err := fillChildren(c)
		if err != nil {
			return err
		}

		childrenNos = append(childrenNos, c.GetNo())
		if exists := childrenNoMap[c.GetNo()]; exists != nil {
			return fmt.Errorf("[%s][%s]包含重复的顺序编号[%d]", exists.GetID(), c.GetID(), c.GetNo())
		}
		childrenNoMap[c.GetNo()] = c
	}

	sort.Ints(childrenNos)

	childrenHTML := make([]string, len(childrenNos))
	childrenCSS := make([]string, len(childrenNos))
	childrenJS := make([]string, len(childrenNos))

	for i, ID := range childrenNos {

		childIndex := fmt.Sprintf("%d", i)

		childrenHTML[i] = replace(childrenNoMap[ID].GetHTML(), RESERVED_CHILD_INDEX, childIndex)
		childrenCSS[i] = replace(childrenNoMap[ID].GetCSS(), RESERVED_CHILD_INDEX, childIndex)
		childrenJS[i] = replace(childrenNoMap[ID].GetJS(), RESERVED_CHILD_INDEX, childIndex)
	}

	html := replace(comp.GetHTML(), RESERVED_CHILDREN, strings.Join(childrenHTML, BREAK))
	if html == comp.GetHTML() {
		html += BREAK + strings.Join(childrenHTML, BREAK)
	}
	comp.SetHTML(html)

	css := replace(comp.GetCSS(), RESERVED_CHILDREN, strings.Join(childrenCSS, BREAK))
	if css == comp.GetCSS() {
		css += BREAK + strings.Join(childrenCSS, BREAK)
	}
	comp.SetCSS(css)

	js := replace(comp.GetJS(), RESERVED_CHILDREN, strings.Join(childrenJS, BREAK))
	if js == comp.GetJS() {
		js += BREAK + strings.Join(childrenJS, BREAK)
	}
	comp.SetJS(js)

	return nil
}

func ParseTemplate(comps []IComponent) ([]IComponent, error) {

	commonMap := make(map[string]IComponent, 0)
	idMap := make(map[string]IComponent, 0)

	for _, comp := range comps {
		idMap[comp.GetID()] = comp

		if comp.GetModel() == nil {
			return nil, fmt.Errorf("[%s]找不到模型", comp.GetID())
		}
		if _, exists := commonMap[comp.GetModel().GetModelID()]; !exists {
			commonMap[comp.GetModel().GetModelID()] = comp
		}
	}

	for _, comp := range comps {
		if comp.GetParentID() != "" {
			if p, exists := idMap[comp.GetParentID()]; exists {
				c := p.GetChildren()
				if c == nil {
					c = make([]IComponent, 0)
				}
				p.SetChildren(append(c, comp))
			} else {
				return nil, fmt.Errorf("[%s]找不到父节点[%s]", comp.GetID(), comp.GetParentID())
			}
		}
	}

	for _, comp := range comps {
		comp.SetHTML(comp.GetModel().GetHTMLTemplate())
		comp.SetCSS(comp.GetModel().GetCSSTemplate())
		comp.SetJS(comp.GetModel().GetJSTemplate())
		err := fillParam(comp, idMap, false)
		if err != nil {
			return nil, err
		}
	}

	result := make([]IComponent, 0)

	for _, comp := range idMap {
		if comp.GetParentID() != "" {
			continue
		}
		err := fillChildren(comp)
		if err != nil {
			return nil, err
		}
		result = append(result, comp)
	}

	return result, nil
}
