# linkreaper

Checks which URLs within a file are still alive.

- For now, only works with text files.
- This project was made to experiment with go's concurrency.

## Installation

1. Build the binary:
   ```bash
   make build
   ```
2. Install it to `/usr/local/bin`:
   ```bash
   sudo make install
   ```

## Usage

```bash
linkreaper -f <file> -w <numOfWorkers>
```
