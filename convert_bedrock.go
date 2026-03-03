package main

import (
	"context"
	"log"

	"github.com/ntaku256/go-bedrock-nbt-api/api"
	bedrocknbt "github.com/ntaku256/go-bedrock-nbt-converter"
)

// convertBedrockMcworld handles .mcworld file conversion through the
// go-bedrock-nbt-converter library.
func convertBedrockMcworld(ctx context.Context, tmpFile string, params *api.ConvertFileParams) ([]byte, error) {
	opts := &bedrocknbt.ConvertOptions{
		MinX:      int32(valOrDefault(params.MinX, -100)),
		MaxX:      int32(valOrDefault(params.MaxX, 100)),
		MinY:      int32(valOrDefault(params.MinY, -64)),
		MaxY:      int32(valOrDefault(params.MaxY, 320)),
		MinZ:      int32(valOrDefault(params.MinZ, -100)),
		MaxZ:      int32(valOrDefault(params.MaxZ, 100)),
		Dimension: int32(valOrDefault(params.Dimension, 0)),
	}

	nbtBytes, _, _, _, err := bedrocknbt.ConvertMcworld(tmpFile, opts)
	if err != nil {
		log.Printf("Bedrock mcworld conversion error: %v", err)
		return nil, err
	}
	return nbtBytes, nil
}

// convertBedrockMcstructure handles .mcstructure file conversion through
// the go-bedrock-nbt-converter library.
func convertBedrockMcstructure(ctx context.Context, tmpFile string) ([]byte, error) {
	nbtBytes, _, _, _, err := bedrocknbt.ConvertMcstructure(tmpFile)
	if err != nil {
		log.Printf("Bedrock mcstructure conversion error: %v", err)
		return nil, err
	}
	return nbtBytes, nil
}

func valOrDefault(ptr *int, def int) int {
	if ptr == nil {
		return def
	}
	return *ptr
}
