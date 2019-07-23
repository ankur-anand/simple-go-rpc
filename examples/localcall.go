package main


import "fmt"

type User struct {
	Name string
	Age int
}

var userDB = map[int]User{
	1: User{"Ankur", 85},
	9: User{"Anand", 25},
	8: User{"Ankur Anand", 27},
}


func QueryUser(id int) (User, error) {
	if u, ok := userDB[id]; ok {
		return u, nil
	}

	return User{}, fmt.Errorf("id %d not in user db", id)
}


func main() {
	u , err := QueryUser(8)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("name: %s, age: %d \n", u.Name, u.Age)
}