package main

import (
	"log"

	javanbt "github.com/ntaku256/go-java-nbt-converter"
)

// convertJavaNbt converts a Java schematic file (.litematic / .schem / .nbt)
// to a gzip-compressed Java Edition Structure NBT byte slice.
//
// This is now a thin wrapper around the go-java-nbt-converter library.
func convertJavaNbt(tmpFile string) ([]byte, error) {
	nbtBytes, err := javanbt.ConvertAny(tmpFile)
	if err != nil {
		log.Printf("Java conversion error for %s: %v", tmpFile, err)
		return nil, err
	}
	return nbtBytes, nil
}
