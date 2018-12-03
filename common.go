package main

// Common structs
type Config struct {
	Addr     string `json:"addr"`
	RestAddr string `json:"restaddr"`
	LogFile  string `json:"logfile"`
}

type Message struct {
	From    string `json:"from"`
	Content string `json:"content"`
}
