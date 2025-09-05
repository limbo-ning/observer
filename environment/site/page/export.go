package page

import (
	"database/sql"
	"io/ioutil"
	"log"
	"os"
	"regexp"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/site/page/template"
)

var phpReg *regexp.Regexp

func init() {
	phpReg = regexp.MustCompile("<\\?php[\\r\\n\\s].*\\?>")
}

func isPHP(c string) bool {
	return phpReg.MatchString(c)
}

func fillModels(siteID string, comps []template.IComponent) error {
	modelIDs := make([]string, 0)
	models := make(map[string]*Model)

	for _, comp := range comps {
		modelIDs = append(modelIDs, comp.GetModelID())
	}

	models, err := GetSiteModels(siteID, modelIDs...)
	if err != nil {
		return err
	}

	for _, comp := range comps {
		comp.SetModel(models[comp.GetModelID()])
	}

	return nil
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

func writeFile(data []byte, folder, filename string) error {
	if !isExist(folder) {
		if err := os.MkdirAll(folder, os.ModePerm); err != nil {
			return err
		}
	}
	return ioutil.WriteFile(folder+filename, data, os.ModePerm)
}

func ExportSite(siteID string) error {

	return datasource.Txn(func(txn *sql.Tx) {
		editingPages, err := getPages(siteID, txn, true, STATUS_EDIT)
		if err != nil {
			panic(err)
		}

		if len(editingPages) == 0 {
			log.Println("no editing pages to export")
			return
		}

		exportPages := make(map[string]string)
		comps := make([]template.IComponent, 0)

		currentPages, err := getPages(siteID, txn, true, STATUS_PUBLISHED)
		if err != nil {
			panic(err)
		}

		for _, p := range currentPages {
			exportPages[p.(*PageComponent).PageID] = STATUS_PUBLISHED
		}

		for _, p := range editingPages {
			pc := p.(*PageComponent)

			if pc.ModelID == "" && pc.ComponentID == "deleted" {
				delete(exportPages, pc.PageID)
			} else {
				exportPages[pc.PageID] = STATUS_EDIT
			}
		}

		pages := make([]template.IComponent, 0)
		for pageID, status := range exportPages {
			pageComponents, err := getPageComponents(siteID, txn, true, pageID, status)
			if err != nil {
				panic(err)
			}
			comps = append(comps, pageComponents...)

			if err := fillModels(siteID, pageComponents); err != nil {
				panic(err)
			}

			exportedPage, err := template.ParseTemplate(pageComponents)
			if err != nil {
				panic(err)
			}

			pages = append(pages, exportedPage...)
		}

		folder := Config.PageExportPath + siteID + "_publishing/"
		for _, page := range pages {
			pageID := page.(*PageComponent).PageID
			if err := writeFile([]byte(page.GetJS()), folder+"js/", pageID+".js"); err != nil {
				panic(err)
			}
			if err := writeFile([]byte(page.GetCSS()), folder+"css/", pageID+".css"); err != nil {
				panic(err)
			}
			var htmlSubfix string
			if isPHP(page.GetHTML()) {
				htmlSubfix = ".php"
			} else {
				htmlSubfix = ".html"
			}
			if err := writeFile([]byte(page.GetHTML()), folder, pageID+htmlSubfix); err != nil {
				panic(err)
			}
		}

		if err := os.RemoveAll(Config.PageExportPath + siteID); err != nil {
			panic(err)
		}
		if err := os.Rename(Config.PageExportPath+siteID+"_publishing", Config.PageExportPath+siteID); err != nil {
			panic(err)
		}

		currentComps, err := getPageComponents(siteID, txn, true, "", STATUS_PUBLISHED)
		if err != nil {
			panic(err)
		}

		archivedComps, err := getPageComponents(siteID, txn, true, "", STATUS_ARCHIVED)
		if err != nil {
			panic(err)
		}

		for _, archive := range archivedComps {
			if err := archive.(*PageComponent).delete(siteID, txn); err != nil {
				panic(err)
			}
		}

		for _, current := range currentComps {
			c := current.(*PageComponent)
			c.Status = STATUS_ARCHIVED

			if err := c.update(siteID, txn); err != nil {
				panic(err)
			}
		}

		for _, comp := range comps {
			c := comp.(*PageComponent)

			switch c.Status {
			case STATUS_EDIT:
				c.Status = STATUS_PUBLISHED
				if err := c.update(siteID, txn); err != nil {
					panic(err)
				}
			default:
				c.Status = STATUS_PUBLISHED
				if err := c.insert(siteID, txn); err != nil {
					panic(err)
				}
			}
		}

	})
}

func RenderPage(siteID, pageID, status string) ([]template.IComponent, error) {
	comps, err := getPageComponents(siteID, nil, false, pageID, status)
	if err != nil {
		return nil, err
	}

	if err := fillModels(siteID, comps); err != nil {
		return nil, err
	}

	return template.ParseTemplate(comps)
}
