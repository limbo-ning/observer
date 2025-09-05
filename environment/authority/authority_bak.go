package authority

// import (
// 	"database/sql"
// 	"errors"
// 	"fmt"
// 	"log"

// 	"obsessiontech/common/datasource"
// 	"obsessiontech/environment/relation"
// )

// const AUTH_OWNER = "OWNER"
// const AUTH_MANAGER = "MANAGER"
// const AUTH_OPERATER = "OPERATOR"
// const AUTH_DEVELOPER = "DEVELOPER"

// type Authority struct {
// 	Authority   string `json:"authority"`
// 	Name        string `json:"name"`
// 	Description string `json:"description"`
// }

// var loading chan byte
// var authorities map[string]*Authority

// func init() {
// 	loading = make(chan byte, 1)
// }

// func LoadAuthority(authority string) (*Authority, error) {
// 	if authorities == nil {
// 		loading <- 1
// 		if authorities == nil {
// 			authorities = make(map[string]*Authority)
// 			rows, err := datasource.GetConn().Query(`
// 				SELECT
// 					authority, name, description
// 				FROM
// 					authority
// 			`)
// 			if err != nil {
// 				log.Println("error load authority: ", err)
// 				panic(err)
// 			}
// 			defer rows.Close()

// 			for rows.Next() {
// 				var a Authority
// 				rows.Scan(&a.Authority, &a.Name, &a.Description)

// 				authorities[a.Authority] = &a
// 			}
// 		}
// 		<-loading
// 	}

// 	if auth, exists := authorities[authority]; exists {
// 		return auth, nil
// 	}

// 	return nil, errors.New("invalid authority")
// }

// func CheckAuthority(siteID string, uid int, authority string) bool {

// 	if siteID == "" || uid <= 0 {
// 		return false
// 	}

// 	return CheckSiteAuthority("c", uid, "site", siteID, authority)
// }

// func CheckSiteAuthority(siteID string, uid int, target, targetID, authority string) bool {
// 	if siteID == "" || uid <= 0 {
// 		return false
// 	}

// 	if relations, err := relation.ExistRelations(siteID, "user", target, "user", authority, fmt.Sprintf("%d", uid), targetID); err != nil || len(relations) == 0 {
// 		return false
// 	}
// 	return true
// }

// func GrantOwner(siteID string, uid int) error {
// 	LoadAuthority(AUTH_OWNER)

// 	relations, err := relation.ExistRelations("c", "user", "site", "site", AUTH_OWNER, siteID)
// 	if err != nil {
// 		return err
// 	}

// 	return datasource.Txn(func(txn *sql.Tx) {
// 		for _, old := range relations {
// 			if err := old.Delete(siteID, txn); err != nil {
// 				panic(err)
// 			}
// 		}
// 		for authority := range authorities {
// 			r := &relation.Relation{
// 				A:    "user",
// 				B:    "site",
// 				AID:  fmt.Sprintf("%d", uid),
// 				BID:  siteID,
// 				Type: authority,
// 			}

// 			if err := r.Add("c", txn); err != nil {
// 				panic(err)
// 			}
// 		}
// 	})
// }

// func GrantAuthority(siteID string, uid int, authority string) error {
// 	a, _ := LoadAuthority(authority)

// 	if a == nil {
// 		return fmt.Errorf("无效的权限:%s", authority)
// 	}
// 	r := &relation.Relation{
// 		A:    "user",
// 		B:    "site",
// 		AID:  fmt.Sprintf("%d", uid),
// 		BID:  siteID,
// 		Type: authority,
// 	}

// 	return datasource.Txn(func(txn *sql.Tx) {
// 		r.Add("c", txn)
// 	})
// }

// func WithdrawAuthority(siteID string, uid int, authority string) error {
// 	a, _ := LoadAuthority(authority)

// 	if a == nil {
// 		return fmt.Errorf("无效的权限:%s", authority)
// 	}

// 	return datasource.Txn(func(txn *sql.Tx) {

// 		if authority == AUTH_MANAGER {
// 			managers, err := relation.ExistRelationsWithTxn("c", txn, "user", "site", "site", AUTH_MANAGER, siteID)
// 			if err != nil {
// 				panic(err)
// 			}

// 			if len(managers) == 1 {
// 				panic(errors.New("至少需要一名管理者"))
// 			}
// 		}

// 		relations, err := relation.ExistRelationsWithTxn("c", txn, "user", "site", "user", authority, fmt.Sprintf("%d", uid), siteID)
// 		if err != nil {
// 			panic(err)
// 		}

// 		if len(relations) == 1 {
// 			if err := relations[0].Delete("c", txn); err != nil {
// 				panic(err)
// 			}
// 		}
// 	})
// }

// func GetAuthoritis(siteID string, uid int) ([]*Authority, error) {

// 	relations, err := relation.ExistRelations("c", "user", "site", "user", "", fmt.Sprintf("%d", uid), siteID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	result := make([]*Authority, 0)
// 	for _, r := range relations {
// 		if auth, err := LoadAuthority(r.Type); err != nil {
// 			log.Println("error load authority: ", err)
// 		} else {
// 			result = append(result, auth)
// 		}
// 	}

// 	return result, nil
// }
