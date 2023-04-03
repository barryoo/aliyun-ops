package main

import (
	"log"

	gr "github.com/awesome-fc/golang-runtime"
	"github.com/joho/godotenv"
)

func initialize(ctx *gr.FCContext) error {
	ctx.GetLogger().Infoln("init golang!")
	return nil
}

func main() {
	gr.Start(handler, initialize)
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("Error loading .env file, err: %s", err)
		panic(err)
	}
}
