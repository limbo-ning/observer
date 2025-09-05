package bill

import (
	"fmt"
	"math"
	"os/exec"
	"strconv"
	"strings"

	"obsessiontech/common/datasource"
	"obsessiontech/environment/resource"
	"obsessiontech/environment/site/page"
)

func GetSiteStorage(siteID string) (int, error) {
	var count int

	var dataSize float64
	if err := datasource.GetConn().QueryRow(`
		SELECT 
			SUM(TRUNCATE((data_length + index_length) / 1024 / 1024, 2))
		FROM 
			information_schema.tables
		WHERE 
			table_schema = ? and table_name like ?;
	`, "projectc", siteID+"_%").Scan(&dataSize); err != nil {
		return -1, err
	}

	count += int(math.Ceil(dataSize))

	pageSizeResult, err := exec.Command("bash", "-c", fmt.Sprintf("du -h --max-depth=0 %s%s | awk '{print $1}'", page.Config.PageExportPath, siteID)).Output()
	if err != nil {
		return -1, err
	}

	pageSize, err := strconv.ParseFloat(strings.Replace(string(pageSizeResult), "M", "", 1), 64)
	if err != nil {
		return -1, err
	}

	count += int(math.Ceil(pageSize))

	staticSizeResult, err := exec.Command("bash", "-c", fmt.Sprintf("du -h --max-depth=0 %s%s | awk '{print $1}'", resource.Config.ResourceFolderPath, siteID)).Output()
	if err != nil {
		return -1, err
	}
	staticSize, err := strconv.ParseFloat(strings.Replace(string(staticSizeResult), "M", "", 1), 64)
	if err != nil {
		return -1, err
	}

	count += int(math.Ceil(staticSize))

	return count, nil
}
