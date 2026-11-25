package web

import "embed"

//go:embed static/* templates/*
var embedded embed.FS
