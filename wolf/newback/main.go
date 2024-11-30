package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// dynamic functions
	http.HandleFunc("/create_interface",c_interface)//开始界面跳转创建房间
	http.HandleFunc("/join_interface",j_interface)//开始界面跳转进入房间
	http.HandleFunc("/create", createHandler)//创建房间跳转房间界面
	http.HandleFunc("/operation", operationHandler)
	http.HandleFunc("/play", playHandler)
	http.HandleFunc("/reissue", reissueHandler)
	http.HandleFunc("/room", roomHandler)//处理对房间信息的请求
	http.HandleFunc("/view", viewHandler)
	// static handlers
	http.Handle("/", http.FileServer(http.Dir("C:/Users/86135/Desktop/学习/wolf")))
	http.Handle("/s", http.FileServer(http.Dir("C:/Users/86135/Desktop/学习/wolf/s")))
	http.Handle("/s/images", http.FileServer(http.Dir("C:/Users/86135/Desktop/学习/wolf/s/images")))
	// listen ports
	log.Fatal(http.ListenAndServe(":8080", nil))
}


func Connection() (*sql.DB, error) {
	db, err := sql.Open("mysql", "root:@tcp(127.0.0.1:3306)/gameplay")
	if err != nil {
		return nil, err
	}
	return db, nil
}

// func Connection() (*sql.DB, error) {
// 	db, err := sql.Open("sqlite3", "../gameplay.db3") // 使用 SQLite 数据库文件
// 	if err != nil {
// 		return nil, err
// 	}
// 	return db, nil
// }

func c_interface(w http.ResponseWriter,r *http.Request){
	if r.Method == http.MethodGet {
		http.Redirect(w, r, "create.html", http.StatusFound)
		return
	}else{
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
}

func j_interface(w http.ResponseWriter,r *http.Request){
	if r.Method == http.MethodGet {
		http.Redirect(w, r, "join.html", http.StatusFound)
		return
	}else{
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	post := r.PostForm
	addRole := ""
	roleNo := 0

	for key := range post {
		if key != "n-werewolf" && key != "n-folk" && key != "username" {
			addRole += key + ";"
			roleNo++
		}
	}
	addRole = strings.TrimSuffix(addRole, ";")

	werewolves := post.Get("n-werewolf")
	folks := post.Get("n-folk")
	roomNo := rand.Intn(100)
	creator := post.Get("username")

	wn0, _ := strconv.Atoi(werewolves)
	fn0, _ := strconv.Atoi(folks)
	roleNo += wn0 + fn0

	handle, err := Connection()
	if err != nil {
		http.Error(w, "Connection failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer handle.Close()

	var existingRoom int
	err = handle.QueryRow("SELECT RmNo FROM Game WHERE RmNo=?", roomNo).Scan(&existingRoom)
	if err != nil {
		fmt.Println(err.Error())
	}

	if err == nil {
		_, _ = handle.Exec("DELETE FROM Game WHERE RmNo=?", roomNo)
		_, _ = handle.Exec("DELETE FROM Running WHERE RmNo=?", roomNo)
		_, _ = handle.Exec("DELETE FROM Player WHERE RmNo=?", roomNo)
		// Room exists, deletion complete
	} else {
		_, err := handle.Exec("INSERT INTO Game VALUES(?, ?, ?, ?, ?, ?, '0', 0, ?)", roomNo, addRole, werewolves, folks, roleNo, creator, creator)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = handle.Exec("INSERT INTO Running (RmNo, death1, death2, death3, start) VALUES(?, 0, 0, 0, 0)", roomNo)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for key := range post {
			if key != "n-werewolf" && key != "n-folk" && key != "username" {
				_, _ = handle.Exec("UPDATE Running SET ??=0 WHERE RmNo=?", key, roomNo)
			}
		}
	}

	fmt.Fprintf(w, "<script>location.href='room.html?room=%d&user=%s'</script>", roomNo, creator)
}

func operationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	post := r.Form
	role := post.Get("b-info")
	self := post.Get("b-self")
	room := post.Get("b-room")
	preRole := post.Get("b-prev")

	handle, err := Connection()
	if err != nil {
		http.Error(w, "Connection failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer handle.Close()

	switch role {
	case "w":
		for key := range post {
			if key != "b-info" && key != "flip-checkbox" && key != "b-room" {
				id := post.Get(key)[6:] // Assuming you meant to retrieve everything after the first 6 characters
				updateString := fmt.Sprintf("UPDATE Running SET death1=%s, w=%s, \"%s\"=-\"%s\" WHERE RmNo=%s", id, id, preRole, preRole, room)
				_, err := handle.Exec(updateString)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				http.Redirect(w, r, fmt.Sprintf("room.html?room=%s&user=%s", room, self), http.StatusSeeOther)
				return
			}
		}

	case "g-witch":
		result1, result2, result3 := 0, 0, 0

		queryString := fmt.Sprintf("UPDATE Running SET \"g-witch\"=100, \"%s\"=-\"%s\" WHERE RmNo=%s", preRole, preRole, room)
		_, err = handle.Exec(queryString)
		if err != nil {
			http.Error(w, "error", http.StatusInternalServerError)
		}
		result3 = 1

		if board := post.Get("board"); board != "" {
			id := board[6:]
			_, err := handle.Exec("UPDATE Running SET death2=? WHERE RmNo=?", id, room)
			if err == nil {
				result2 = 1
			}
		}

		if post.Get("flip-checkbox") == "on" {
			_, err := handle.Exec("UPDATE Running SET \"g-witch\"=death1, death1=0 WHERE RmNo=?", room)
			if err == nil {
				result1 = 1
			}
		}

		if result1 > 0 || result2 > 0 || result3 > 0 {
			http.Redirect(w, r, fmt.Sprintf("room.html?room=%s&user=%s", room, self), http.StatusSeeOther)
		} else {
			http.Error(w, "error", http.StatusInternalServerError)
		}

	case "g-seer":
		board := post.Get("board")
		queryString := fmt.Sprintf("UPDATE Running SET \"%s\"=-\"%s\", \"g-seer\"=%s WHERE RmNo==%s", preRole, preRole, board, room)
		_, err = handle.Exec(queryString)
		if err == nil {
			fmt.Fprintln(w, "success")
		} else {
			http.Error(w, "error", http.StatusInternalServerError)
		}

	case "g-guard":
		var kill, save int
		row := handle.QueryRow("SELECT death1, \"g-witch\" FROM Running WHERE RmNo=?", room)
		if err := row.Scan(&kill, &save); err != nil {
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
		var rowsAffected int64
		rowsAffected = 0
		var qResult sql.Result
		var id int
		deathInfo := ""

		if board := post.Get("board"); board != "" {
			id, _ = strconv.Atoi(board[6:]) // 转换 string 到 int
			if kill != 0 && id == kill {
				deathInfo = ", death1=0"
			} else if kill == 0 && save == id {
				deathInfo = ", death1=" + strconv.Itoa(id)
			} //else here: use the default settings, no change
		} else {
			id = 100
		}

		queryString := fmt.Sprintf("UPDATE Running SET \"g-guard\"=%d, \"%s\"=-\"%s\"%s WHERE RmNo=%s", id, preRole, preRole, deathInfo, room)
		qResult, _ = handle.Exec(queryString)
		rowsAffected, _ = qResult.RowsAffected()

		if rowsAffected > 0 {
			http.Redirect(w, r, fmt.Sprintf("room.html?room=%s&user=%s", room, self), http.StatusSeeOther)
		} else {
			http.Error(w, "error", http.StatusInternalServerError)
		}

	case "w-devil":
		board := post.Get("board")
		queryString := fmt.Sprintf("UPDATE Running SET \"%s\"=-\"%s\", \"w-devil\"=%s WHERE RmNo=%s", preRole, preRole, board, room)
		_, err = handle.Exec(queryString)
		if err == nil {
			fmt.Fprintln(w, "success")
		} else {
			http.Error(w, "error", http.StatusInternalServerError)
		}

	default:
		http.Error(w, "Invalid role", http.StatusBadRequest)
	}
}

func playHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	room := r.URL.Query().Get("room")
	handle, err := Connection()
	if err != nil {
		http.Error(w, "Connection failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer handle.Close()

	if start := r.URL.Query().Get("start"); start != "" {
		result, err1 := handle.Exec("UPDATE Running SET start=1 WHERE RmNo=?", room)
		rowsAffected, err2 := result.RowsAffected()
		if err1 != nil || err2 != nil || rowsAffected == 0 {
			fmt.Fprintln(w, "error")
		} else {
			fmt.Fprintln(w, "success")
		}
	} else if verify := r.URL.Query().Get("verify"); verify != "" {
		role := r.URL.Query().Get("role")
		if role == "" {
			role = "start"
		}
		var prepared sql.NullString
		queryString := fmt.Sprintf("SELECT \"%s\" FROM Running WHERE RmNo=%s", role, room)
		err = handle.QueryRow(queryString).Scan(&prepared)
		if err != nil {
			if err == sql.ErrNoRows {
				fmt.Fprintln(w, "0")
			} else {
				http.Error(w, "play verify error "+err.Error(), http.StatusInternalServerError)
			}
		} else {
			if prepared.Valid {
				fmt.Fprintln(w, prepared.String)
			} else {
				fmt.Fprintln(w, "0")
			}
		}

	} else if acquire3 := r.URL.Query().Get("acquire3"); acquire3 != "" {
		item := r.URL.Query().Get("item")
		item2 := r.URL.Query().Get("item2")
		item3 := r.URL.Query().Get("item3")

		var p1 sql.NullString
		var p2 sql.NullString
		var p3 sql.NullString
		err := handle.QueryRow("SELECT \""+item+"\", \""+item2+"\", \""+item3+"\" FROM Running WHERE RmNo=?", room).Scan(&p1, &p2, &p3) // 应该定义为多个变量
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		} else {
			prepared := p1.String + " " + p2.String + " " + p3.String
			fmt.Fprintln(w, prepared)
		}

	} else if ongoing := r.URL.Query().Get("ongoing"); ongoing != "" {
		rows, err := handle.Query("SELECT * FROM Running WHERE RmNo=?", room)
		if err != nil {
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var prepared []map[string]interface{}
		for rows.Next() {
			var rowData = make(map[string]interface{})
			columns, _ := rows.Columns()
			values := make([]interface{}, len(columns))
			for i := range values {
				values[i] = new(sql.RawBytes)
			}
			err = rows.Scan(values...)
			if err != nil {
				http.Error(w, "error", http.StatusInternalServerError)
				return
			}
			for i, col := range columns {
				b := values[i].(*sql.RawBytes)
				rowData[col] = string(*b)
			}
			prepared = append(prepared, rowData)
		}
		json.NewEncoder(w).Encode(prepared)

	} else {
		http.Error(w, "Invalid request", http.StatusBadRequest)
	}
}

func reissueHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	roomNo := r.URL.Query().Get("room")
	playNo := r.URL.Query().Get("playerNo")

	handle, err := Connection()
	if err != nil {
		http.Error(w, "Connection failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer handle.Close()

	// 角色重新分配
	newArray := make([]int, 0)
	users, err := handle.Query("SELECT * FROM Player WHERE RmNo=?", roomNo)
	if err != nil {
		http.Error(w, "Error fetching players: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer users.Close()

	i := 1
	for users.Next() {
		var username string
		var name string
		var oldRoomNo int
		var role string
		var idNo int
		// Assuming you have a way to get the player number from the row
		err := users.Scan(&username, &name, &oldRoomNo, &role, &idNo) // Adjust according to your Player table structure
		if err != nil {
			http.Error(w, "Error scanning player: "+err.Error(), http.StatusInternalServerError)
			return
		}

		totalPlayers, _ := strconv.Atoi(playNo)

		newRole := rand.Intn(totalPlayers) + 1 // Generate a random role between 1 and playNo
		for contains(newArray, newRole) {
			newRole = rand.Intn(totalPlayers) + 1
		}
		newArray = append(newArray, newRole)

		_, err = handle.Exec("UPDATE Player SET Role=? WHERE RmNo=? AND No=?", newRole, roomNo, i)
		if err != nil {
			http.Error(w, "Error updating player role: "+err.Error(), http.StatusInternalServerError)
			return
		}
		i++
	}

	// 更新 Running 表
	runners, err := handle.Query("SELECT * FROM Running WHERE RmNo=?", roomNo)
	if err != nil {
		http.Error(w, "Error fetching running data: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer runners.Close()

	if runners.Next() {
		columns, err := runners.Columns()
		if err != nil {
			http.Error(w, "Error fetching columns: "+err.Error(), http.StatusInternalServerError)
			return
		}

		for _, key := range columns {
			if key != "RmNo" {
				_, err = handle.Exec("UPDATE Running SET \""+key+"\"=0 WHERE RmNo=?", roomNo)
				if err != nil {
					http.Error(w, "Error updating running data: "+err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
	}

	// Redirect back to the previous page
	fmt.Fprintln(w, "<script>window.history.back();</script>")
}

func roomHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	room := r.URL.Query().Get("room")
	user := r.URL.Query().Get("user")

	response := map[string]interface{}{
		"selfRole":   -2,
		"playerNo":   8,
		"roleList":   []string{},
		"No":         0,
		"playerList": map[int]string{},
		"creator":    "",
	}

	handle, err := Connection()
	if err != nil {
		http.Error(w, "Connection failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer handle.Close()

	var flagHasGame bool
	var setRoles string
	var werewolves, folks int
	var creator string

	gameinfo, err := handle.Query("SELECT PlayerNo,Role,Werewolf,Folk,Creator FROM Game WHERE RmNo=?", room)
	if err != nil {
		http.Error(w, "Error fetching game info: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer gameinfo.Close()

	//房间的玩家数量限制和房主
	for gameinfo.Next() {
		var playerNo int
		err := gameinfo.Scan(&playerNo, &setRoles, &werewolves, &folks, &creator)
		if err != nil {
			http.Error(w, "Error scanning game info: "+err.Error(), http.StatusInternalServerError)
			return
		}
		response["playerNo"] = playerNo
		response["creator"] = creator
		flagHasGame = true
	}

	if !flagHasGame {
		json.NewEncoder(w).Encode(response)
		return
	}

	//房间的角色列表
	roles := strings.Split(setRoles, ";")
	for i := 0; i < werewolves; i++ {
		roles = append(roles, "w")
	}
	for i := 0; i < folks; i++ {
		roles = append(roles, "f")
	}
	response["roleList"] = roles

	playerinfo, err := handle.Query("SELECT Role, No FROM Player WHERE RmNo=? AND Username=?", room, user)
	if err != nil {
		http.Error(w, "Error fetching player info: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer playerinfo.Close()

	//该玩家担任的角色以及局内编号
	for playerinfo.Next() {
		var selfRole, playerNo int
		err := playerinfo.Scan(&selfRole, &playerNo)
		if err != nil {
			http.Error(w, "Error scanning player info: "+err.Error(), http.StatusInternalServerError)
			return
		}
		response["selfRole"] = selfRole
		response["No"] = playerNo
	}

	listInfo, err := handle.Query("SELECT Username, No FROM Player WHERE RmNo=?", room)
	if err != nil {
		http.Error(w, "Error fetching player list: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer listInfo.Close()

	//局内其他玩家的名称以及局内编号
	for listInfo.Next() {
		var username string
		var no int
		err := listInfo.Scan(&username, &no)
		if err != nil {
			http.Error(w, "Error scanning player list: "+err.Error(), http.StatusInternalServerError)
			return
		}
		response["playerList"].(map[int]string)[no] = username
	}

	//该玩家是新加入房间的情况
	if response["selfRole"].(int) == -2 {
		var role []int
		var maxNo int

		hasRole, err := handle.Query("SELECT Role, No FROM Player WHERE RmNo=?", room)
		if err != nil {
			http.Error(w, "Error fetching roles: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer hasRole.Close()

		for hasRole.Next() {
			var r, no int
			err := hasRole.Scan(&r, &no)
			if err != nil {
				http.Error(w, "Error scanning roles: "+err.Error(), http.StatusInternalServerError)
				return
			}
			role = append(role, r)
			if no > maxNo {
				maxNo = no
			}
		}

		//房间已满
		if len(role) >= response["playerNo"].(int) {
			response["selfRole"] = -1
		} else {									//给该玩家赋予角色
			newRole := rand.Intn(response["playerNo"].(int)) + 1
			for contains(role, newRole) {
				newRole = rand.Intn(response["playerNo"].(int)) + 1
			}
			maxNo++
			_, err = handle.Exec("INSERT INTO Player VALUES (?, '', ?, ?, ?)", user, room, newRole, maxNo)
			if err != nil {
				http.Error(w, "Error inserting player: "+err.Error(), http.StatusInternalServerError)
				return
			}

			response["selfRole"] = newRole
			response["No"] = maxNo
		}
	}

	json.NewEncoder(w).Encode(response)
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	room := r.URL.Query().Get("room")
	//role := r.URL.Query().Get("role")
	id := r.URL.Query().Get("board")

	handle, err := Connection()
	if err != nil {
		http.Error(w, "Connection failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer handle.Close()

	var ms string
	query := "SELECT Role FROM Player WHERE RmNo=? AND No=?"
	err = handle.QueryRow(query, room, id).Scan(&ms)
	if err != nil {
		http.Error(w, "Error fetching role: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the role found
	fmt.Fprintln(w, ms)
}

func contains(slice []int, item int) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}
