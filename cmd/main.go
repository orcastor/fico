package main

import (
    "flag"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/orcastor/fico"
)

var (
    inputPath  string
    outputPath string
    format     string
    width      int
    height     int
    index      int
    indexSet   bool
)

func main() {
    // Parse command-line arguments
    flag.StringVar(&inputPath, "input", "", "Path to the input file / directory")
    flag.StringVar(&outputPath, "output", "", "Path to the output file (optional)")
    flag.StringVar(&format, "format", "png", "Output format")
    flag.IntVar(&width, "width", 32, "Image width")
    flag.IntVar(&height, "height", 32, "Image height")
    flag.IntVar(&index, "index", 0, "Image index (optional)")

    flag.Parse()

    // Check if required arguments are provided
    if inputPath == "" {
        fmt.Println("The input parameter is required")
        flag.Usage()
        os.Exit(1)
    }

    // Derive output path if not provided
    if outputPath == "" {
        baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
        outputPath = baseName + "." + format
    }

    // Create output file
    outputFile, err := os.Create(outputPath)
    if err != nil {
        fmt.Printf("Failed to create output file: %v\n", err)
        os.Exit(1)
    }
    defer outputFile.Close()
    
     // Get information from GetInfo function
    info, err := fico.GetInfo(inputPath)
    if err != nil {
        fmt.Printf("Error getting info: %v\n", err)
        os.Exit(1)
    }

    // Use retrieved information
    if info.IconIndex != nil {
        index = *info.IconIndex
        indexSet = true
    }

    // Check if the index flag was set
    flag.Visit(func(f *flag.Flag) {
        if f.Name == "index" {
            indexSet = true
        }
    })

    // Prepare configuration
    var indexPtr *int
    if indexSet {
        indexPtr = &index
    }
    config := fico.Config{
        Format: format,
        Index:  indexPtr,
        Width:  width,
        Height: height,
    }

    // Call fico.F2ICO function
    err = fico.F2ICO(outputFile, inputPath, config)
    if err != nil {
        fmt.Printf("Error converting icon: %v\n", err)
        os.Exit(1)
    }

    fmt.Printf("%s -> %s\n", inputPath, outputPath)
    fmt.Println("Icon conversion successful")
}
