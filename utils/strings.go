package utils

import gonanoid "github.com/matoous/go-nanoid"

func GetNanoid() (nanoid string) {
	nanoid, _ = gonanoid.Nanoid()
	return
}
