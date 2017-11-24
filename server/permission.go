package server

import (
	"fmt"
	"database/sql"
	"github.com/go-sql-driver/mysql"
)

//127.0.0.1:8080/permissions?email=cuiwenchang@k2data.com.cn

type UserPolicy struct {
	ResourceId string	`json:"resource_id"`
	PolicyName string `json:"policy_name"`
}
func (s Server)permissionGetByEmail(userEmail string) map[string][]string {
	newConfig := mysql.NewConfig()
	newConfig.Net = "tcp"
	newConfig.Addr = s.sqlConf.Addr
	newConfig.User = s.sqlConf.User
	newConfig.Passwd = s.sqlConf.Password
	newConfig.DBName = s.sqlConf.Password
	dsn := newConfig.FormatDSN()
	fmt.Println(dsn)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	defer db.Close()

	sqlStr := fmt.Sprintf("select name,resource_id from policy WHERE id in (select policy_id from permission_user, user where user.id=permission_user.user_id and user.email='%s');",userEmail)
	fmt.Println(sqlStr)
	rows, err := db.Query(sqlStr)
	if err != nil{
		fmt.Println(err.Error())
		return nil
	}
	cols, err := rows.Columns()
	fmt.Println(cols)

	var policies []UserPolicy
	for rows.Next(){
		var name string
		var resource_id string

		rows.Scan(&name, &resource_id)
		fmt.Println(name, resource_id)
		policies = append(policies, UserPolicy{resource_id,name})
	}
	var perMap =make(map[string][]string)
	for _,policy := range policies{
		perMap[policy.ResourceId] = append(perMap[policy.ResourceId], policy.PolicyName)
	}
	fmt.Println(perMap)
	return perMap
}
