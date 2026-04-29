package router

import (
	"os"
	"fmt"
	"net"
)

type Page struct {
	Title string
	Body  []byte
}

func loadPage(title string) *Page, err {
	filename := title + ".txt"
	body, _ := os.ReadFile(filename)
	if err!=null{
		return nil, err
	}
	return &Page{Title: title, Body: body}
}
