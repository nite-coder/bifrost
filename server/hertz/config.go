package main

type Route struct {
	Match    string
	Method   []string
	Entry    []string
	Upstream string
}
