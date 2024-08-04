# Sitemap Generator

This Cli tool extracts links from a specified URL and writes them to a sitemap XML file. The generator handles concurrent requests and limits the number of links it processes. It also uses custom flags for specifying the maximum number of links and the output file.

## Features

- Concurrent link extraction using goroutines
- Sitemap XML file generation
- Logging for progress and errors

## Requirements

- Go 1.18 or later
- Go modules enabled (`go mod`)

## Installation

1. Clone the repository:
   ```sh
   git clone https://github.com/yourusername/sitemap-generator.git
   cd sitemap-generator
   ```
2. Build the project:
    ```sh
    go build -o sitemap-generator

3. Run the project:
    ```sh
    ./sitemap-generator -t <target-url> -maxLinks <number-of-links> -o <output-file>
    ```
