package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"sync"
)

var clientsMu sync.Mutex

type User struct {
	Name         string `json:"name"`
	PasswordHash string `json:"hash"`
}

func getHash(text string) string {
	hasher := sha256.New()
	hasher.Write([]byte(text))
	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash)
}

func readUsersFromFile(filename string) ([]User, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var users []User
	err = json.Unmarshal(bytes, &users)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func writeUsers(filename string, users []User) error {
	jsonData, err := json.Marshal(users)
	if err != nil {
		fmt.Println("Ошибка создания jsonData")
		return err
	}
	err = ioutil.WriteFile(filename, jsonData, 0644)
	if err != nil {
		fmt.Println("Ошибка записи")
		return err
	}
	return nil
}

var activeClients = make(map[net.Conn]string)

func main() {
	filename := "clients.json"

	users, err := readUsersFromFile(filename)
	fmt.Println(users)

	//user := User{"1", getHash("pass")}
	//
	//users = append(users, user)
	//
	//fmt.Println(users)
	//_ = writeUsers(filename, users)

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Ошибка запуска сервера: ", err)
		return
	}
	defer listener.Close()
	fmt.Println("Сервер запущен!")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Ошибка установления соединения: ", err)
			continue
		}
		go handleConnection(conn, &users, filename)
	}
}

func userInUsers(username string, users *[]User) (string, bool) {
	for _, user := range *users {
		if username == user.Name {
			return user.PasswordHash, true
		}
	}
	return "1", false
}

func handleConnection(conn net.Conn, users *[]User, filename string) {
	defer conn.Close()

	//conn.Write([]byte("Enter your username: "))
	username := ""
	password := ""
	for len(username) == 0 {
		conn.Write([]byte("Введите имя пользователя\n"))
		username, _ = bufio.NewReader(conn).ReadString('\n')
		username = strings.TrimSpace(username)
	}
	for len(password) == 0 {
		conn.Write([]byte(username + ", введите пароль" + "!\n"))
		password, _ = bufio.NewReader(conn).ReadString('\n')
		password = strings.TrimSpace(password)
	}

	trueHash, flag := userInUsers(username, users)

	if !flag {
		clientsMu.Lock()
		user := User{username, getHash(password)}
		*users = append(*users, user)
		_ = writeUsers(filename, *users)
		clientsMu.Unlock()
	} else {
		//hash := getHash(password)
		for getHash(password) != trueHash {
			conn.Write([]byte("Неверный пароль, попробуйте ещё раз, " + username + "!\n"))
			password, _ := bufio.NewReader(conn).ReadString('\n')
			password = strings.TrimSpace(password)
		}
	}

	activeClients[conn] = username

	conn.Write([]byte("Welcome, " + username + "!\n"))

	for {
		message, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			delete(activeClients, conn)
			return
		}
		broadcastMessage(username + ": " + message)
	}
}

func broadcastMessage(message string) {
	for conn := range activeClients {
		conn.Write([]byte(message))
	}
}
