package main

import (
	"log"

	bedrocknbt "github.com/ntaku256/go-bedrock-nbt-converter"
)

// convertBedrockMcworld handles .mcworld file conversion through the
// go-bedrock-nbt-converter library.
func convertBedrockMcworld(tmpFile string, params map[string]string) ([]byte, error) {
	opts := &bedrocknbt.ConvertOptions{
		MinX:      parseInt32Param(params, "min_x", -100),
		MaxX:      parseInt32Param(params, "max_x", 100),
		MinY:      parseInt32Param(params, "min_y", -64),
		MaxY:      parseInt32Param(params, "max_y", 320),
		MinZ:      parseInt32Param(params, "min_z", -100),
		MaxZ:      parseInt32Param(params, "max_z", 100),
		Dimension: parseInt32Param(params, "dimension", 0),
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
func convertBedrockMcstructure(tmpFile string) ([]byte, error) {
	nbtBytes, _, _, _, err := bedrocknbt.ConvertMcstructure(tmpFile)
	if err != nil {
		log.Printf("Bedrock mcstructure conversion error: %v", err)
		return nil, err
	}
	return nbtBytes, nil
}
