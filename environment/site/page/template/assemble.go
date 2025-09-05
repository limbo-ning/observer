package template

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
)

func assembleParam(comp IComponent, components map[string]IComponent, paramAlias map[string]string) error {

	sameNames := make(map[string]string)

	for origin, alias := range paramAlias {

		log.Println("assemble param: ", origin, alias)

		if origin == RESERVED_CHILDREN {
			continue
		}

		if origin == alias {
			alias = fmt.Sprintf("%s%d", origin, len(sameNames))
			sameNames[origin] = alias
			// return errors.New("重命名参数不能与之前一样：" + origin)
			// continue
		}

		rule := fmt.Sprintf("((%s)%s(:\\S)?)|(%s(:\\S+)?(%s))", PLACEHOLDER_PREFIX, origin, origin, PLACEHOLDER_SUBFIX)
		matcher := regexp.MustCompile(rule)

		comp.SetHTML(matcher.ReplaceAllString(comp.GetHTML(), fmt.Sprintf("${2}%s${3}${5}${6}", alias)))
		comp.SetCSS(matcher.ReplaceAllString(comp.GetCSS(), fmt.Sprintf("${2}%s${3}${5}${6}", alias)))
		comp.SetJS(matcher.ReplaceAllString(comp.GetJS(), fmt.Sprintf("${2}%s${3}${5}${6}", alias)))
		log.Println("js replace: ", comp.GetJS())
	}

	if err := fillParam(comp, components, true); err != nil {
		return err
	}

	for origin, alias := range sameNames {

		log.Println("assemble param same name: ", origin, alias)

		rule := fmt.Sprintf("((%s)%s(:\\S)?)|(%s(:\\S+)?(%s))", PLACEHOLDER_PREFIX, alias, alias, PLACEHOLDER_SUBFIX)
		matcher := regexp.MustCompile(rule)

		comp.SetHTML(matcher.ReplaceAllString(comp.GetHTML(), fmt.Sprintf("${2}%s${3}${5}${6}", origin)))
		comp.SetCSS(matcher.ReplaceAllString(comp.GetCSS(), fmt.Sprintf("${2}%s${3}${5}${6}", origin)))
		comp.SetJS(matcher.ReplaceAllString(comp.GetJS(), fmt.Sprintf("${2}%s${3}${5}${6}", origin)))
	}

	return nil
}

func assembleChildren(comp IComponent, paramAlias map[string]map[string]string) error {

	if alias, exists := paramAlias[comp.GetID()]; exists {
		if _, exists := alias[RESERVED_CHILDREN]; exists {
			return nil
		}
	}

	if comp.GetChildren() == nil || len(comp.GetChildren()) == 0 {
		comp.SetHTML(replace(comp.GetHTML(), RESERVED_CHILDREN, ""))
		comp.SetCSS(replace(comp.GetCSS(), RESERVED_CHILDREN, ""))
		comp.SetJS(replace(comp.GetJS(), RESERVED_CHILDREN, ""))
		return nil
	}

	childrenNoMap := make(map[int]IComponent, 0)
	childrenNos := make([]int, 0)

	for _, c := range comp.GetChildren() {

		err := assembleChildren(c, paramAlias)
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

func AssembleTemplate(comps []IComponent, paramAlias map[string]map[string]string) (IComponent, error) {

	var root IComponent

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
		} else {
			if root != nil {
				return nil, errors.New("多于一个根节点")
			}
			root = comp
		}
	}

	for _, comp := range comps {

		if comp.GetID() != root.GetID() {
			oldID := comp.GetID()
			newID := fmt.Sprintf("%s%s%s_%s", PLACEHOLDER_PREFIX, RESERVED_ID, PLACEHOLDER_SUBFIX, oldID)

			comp.SetID(newID)
			param := comp.GetParam()
			param[RESERVED_ID] = comp.GetID()
			comp.SetParam(param)

			paramAlias[comp.GetID()] = paramAlias[oldID]
			delete(paramAlias, oldID)
		} else {
			oldID := comp.GetID()
			newID := fmt.Sprintf("%s%s%s", PLACEHOLDER_PREFIX, RESERVED_ID, PLACEHOLDER_SUBFIX)

			comp.SetID(newID)
			param := comp.GetParam()
			param[RESERVED_ID] = comp.GetID()
			comp.SetParam(param)

			paramAlias[comp.GetID()] = paramAlias[oldID]
			delete(paramAlias, oldID)
		}

		comp.SetHTML(comp.GetModel().GetHTMLTemplate())
		comp.SetCSS(comp.GetModel().GetCSSTemplate())
		comp.SetJS(comp.GetModel().GetJSTemplate())
		assembleParam(comp, idMap, paramAlias[comp.GetID()])
	}

	if err := assembleChildren(root, paramAlias); err != nil {
		return nil, err
	}

	return root, nil
}
