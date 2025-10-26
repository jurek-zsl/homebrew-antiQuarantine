# 😷antiQuarantine (aq)

Simple CLI tool for removing the `com.apple.quarantine` extended attribute from files on macOS built on go.

## 📚Platform and Requirements

- Platform: only **macOS**
- Requires: [homebrew](https://brew.sh) installed
- Permissions: Modifying some files may require root privileges
  
## 📥Install

1. You must have [homebrew](https://brew.sh) installed.

2. Installing:
   
- Install using Homebrew (preferred):
  ```
  brew install jurek-zsl/antiquarantine/aq
  ```

- Or add the tap first (alternative):
  ```
  brew tap jurek-zsl/antiquarantine
  brew install aq
  ```

## 📋Usage

> [!TIP]
> I recommend putting filenames and directories in quotes, if they contain spaces or special symbols.


- Show help menu for antiQuarantine
  ```
  aq -h
  ```
- Check for quarantine in a single file:
  ```
  aq `/path/to/file`
  ```
- Remove quarantine from a single file:
  ```
  aq -r `/path/to/file`
  ```
- Check for quarantine in multiple files:
  ```
  aq -f `path`
  ```
- Remove quarantine from multiple files:
  ```
  aq -rf `path`
  ```

## 🏗️Building from source

If you prefer to build locally:

1. Clone the repository:
   ```
   git clone https://github.com/jurek-zsl/homebrew-antiQuarantine.git
   cd homebrew-antiQuarantine
   ```

2. Build (simple approach—adjust if the project uses a specific subdirectory for cmd):
   ```
   go build -o aq
   ```

3. Move the binary into your PATH:
   ```
   sudo mv aq /usr/local/bin/
   ```

## 🔧Troubleshooting

- "Permission denied" errors: try running with `sudo` or adjust file ownership.
- `xattr` not found: ensure you are on macOS and have standard command line tools installed.
- If `aq` doesn't remove the attribute, verify with `xattr -l` and check that the file is not locked or in use.

## ✨Star the project
![GitHub Repo stars](https://img.shields.io/github/stars/jurek-zsl/homebrew-antiQuarantine)
<br>
If you find **antiQuarantine** helpful, please consider giving it a star on GitHub to help others discover it!

## 🤝Contributing

- I'm open on new ideas and improvements, feel free to create an issue or PR.
