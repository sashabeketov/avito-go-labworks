package main

import (
	"fmt"

	"github.com/sashabeketov/avito-go-labworks/internal/config"
)

func main() {
	conf, err := config.LoadConfig()

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println(conf)
}
