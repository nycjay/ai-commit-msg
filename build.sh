#!/bin/bash

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to display help
show_help() {
    echo "AI Commit Message Generator Build Script"
    echo ""
    echo "Usage: ./build.sh [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --version VERSION    Specify a custom version (default: read from VERSION file)"
    echo "  --test               Run unit tests explicitly (tests run by default)"
    echo "  --skip-tests         Skip running tests (tests are run by default)"
    echo "  --test-only          Only run tests without building"
    echo "  --single-platform    Build only for current platform (default: build for all platforms)"
    echo "  --symlink            Create a symlink in /usr/local/bin (macOS/Linux only)"
    echo "  --verbose            Show detailed output during the build process"
    echo "  --help               Show this help message"
    echo ""
    echo "Testing:"
    echo "  By default, the build script automatically runs all unit tests before building."
    echo "  Use --skip-tests to bypass testing if you just want to build quickly."
    echo "  Use --test-only to only run tests without building any binaries."
    echo ""
    echo "Examples:"
    echo "  ./build.sh                   # Run tests and build for all platforms"
    echo "  ./build.sh --version 1.2.3   # Build with a specific version"
    echo "  ./build.sh --skip-tests      # Skip tests and only build"
    echo "  ./build.sh --test-only       # Only run tests without building"
    echo "  ./build.sh --single-platform # Build only for the current platform" 
    echo "  ./build.sh --symlink         # Build and create a symlink in /usr/local/bin"
    echo "  ./build.sh --verbose --test  # Run tests with detailed output"
}

# Default options
VERSION=""
RUN_TESTS=true     # Default to running tests
SKIP_TESTS=false
TEST_ONLY=false
CROSS_COMPILE=true  # Default to cross-platform builds
CREATE_SYMLINK=false # Default to not creating a symlink
VERBOSE=false # Default to non-verbose output

# Check if any arguments were provided - used to determine if this is a default run
NO_ARGS=false
if [[ $# -eq 0 ]]; then
    NO_ARGS=true
fi

# Parse command line arguments
while [[ "$#" -gt 0 ]]; do
    case $1 in
        --version) VERSION="$2"; shift ;;
        --test) RUN_TESTS=true ;;
        --skip-tests) SKIP_TESTS=true ;;
        --test-only) TEST_ONLY=true; RUN_TESTS=true ;;
        --cross-compile) CROSS_COMPILE=true ;;
        --single-platform) CROSS_COMPILE=false ;;
        --symlink) CREATE_SYMLINK=true ;;
        --verbose) VERBOSE=true ;;
        --help) show_help; exit 0 ;;
        *) echo "Unknown parameter passed: $1"; show_help; exit 1 ;;
    esac
    shift
done

# If no version specified, read from VERSION file
if [ -z "$VERSION" ]; then
    VERSION=$(cat VERSION 2>/dev/null || echo "0.1.0")
fi

# Ensure we're in the script's directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# Function to check if Go code can compile
check_compilation() {
    echo -e "${BLUE}Checking if code can compile...${NC}"
    # Use a temporary file to capture any error output
    ERROR_OUTPUT=$(mktemp)
    if ! go build -o /dev/null ./cmd/ai-commit-msg 2>"$ERROR_OUTPUT"; then
        echo -e "${RED}❌ Compilation check failed.${NC}"
        cat "$ERROR_OUTPUT"
        rm "$ERROR_OUTPUT"
        exit 1
    fi
    rm "$ERROR_OUTPUT"
    echo -e "${GREEN}✅ Code compiles successfully.${NC}"
}

# Function to run tests
run_tests() {
    if [ "$SKIP_TESTS" = true ]; then
        echo -e "${YELLOW}Skipping tests as requested.${NC}"
        return 0
    fi

    echo -e "${YELLOW}Running unit tests...${NC}"
    
    # Use a temporary file to capture test output
    TEST_OUTPUT=$(mktemp)
    
    # Run tests with coverage 
    if ! go test -cover ./... > "$TEST_OUTPUT" 2>&1; then
        echo -e "${RED}❌ Tests failed. Stopping build.${NC}"
        cat "$TEST_OUTPUT"
        rm "$TEST_OUTPUT"
        exit 1
    fi
    
    # Count packages, tests, and display coverage information
    PACKAGE_COUNT=$(grep -c "^ok" "$TEST_OUTPUT" || echo 0)
    FAILED_COUNT=$(grep -c "^FAIL" "$TEST_OUTPUT" || echo 0)
    
    echo -e "${BLUE}╔════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║               TEST RESULTS SUMMARY              ║${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════╝${NC}"
    
    # Show test status - make sure we handle the number correctly
    if [ "$FAILED_COUNT" = "0" ]; then
        echo -e "${GREEN}✓ All packages tested successfully${NC}"
        echo -e "${GREEN}✓ No test failures detected${NC}"
    else
        echo -e "${RED}✗ ${FAILED_COUNT} packages had failing tests${NC}"
    fi
    
    echo -e "${BLUE}┌─ Coverage Report ─────────────────────────────┐${NC}"
    
    # Extract and process coverage information
    TOTAL_COV=0
    PACKAGE_WITH_COV=0
    
    # Create a temporary file to store coverage data
    COV_DATA=$(mktemp)
    grep "coverage:" "$TEST_OUTPUT" > "$COV_DATA"
    
    # Count the number of packages with coverage > 0
    PACKAGES_WITH_REAL_COV=0
    TOTAL_COVERAGE=0
    
    # Process each line with coverage information
    # Using direct file reading instead of pipe to avoid subshell variable scope issues
    while read -r line; do
        # Get package name - extract the module part
        if [[ "$line" =~ github\.com/nycjay/ai-commit-msg/([^[:space:]]+) ]]; then
            # Found a package path that matches our repo
            PKG_NAME="${BASH_REMATCH[1]}"
        else
            # Use first field if no match (should not happen often)
            PKG_NAME=$(echo "$line" | awk '{print $1}')
        fi
        
        # Extract coverage percentage 
        if [[ "$line" =~ coverage:\ ([0-9.]+)% ]]; then
            COV="${BASH_REMATCH[1]}"
            
            # Count packages with actual coverage
            if (( $(echo "$COV > 0" | bc -l) )); then
                PACKAGES_WITH_REAL_COV=$((PACKAGES_WITH_REAL_COV + 1))
                TOTAL_COVERAGE=$(echo "$TOTAL_COVERAGE + $COV" | bc)
            fi
            
            # Color-code based on coverage percentage
            if (( $(echo "$COV >= 70" | bc -l) )); then
                COV_COLOR="${GREEN}"
            elif (( $(echo "$COV >= 30" | bc -l) )); then
                COV_COLOR="${YELLOW}"
            else
                COV_COLOR="${RED}"
            fi
            
            # Print package with its coverage
            printf "${BLUE}│${NC} %-30s ${COV_COLOR}%5.1f%%${NC} ${BLUE}│${NC}\n" "$PKG_NAME" "$COV"
        fi
    done < "$COV_DATA"
    
    # Clean up temporary file
    rm -f "$COV_DATA"
    
    # Calculate average coverage if any packages have coverage
    if [ "$PACKAGES_WITH_REAL_COV" -gt 0 ]; then
        AVG_COV=$(echo "scale=1; $TOTAL_COVERAGE / $PACKAGES_WITH_REAL_COV" | bc)
        
        # Color-code the overall coverage
        if (( $(echo "$AVG_COV >= 70" | bc -l) )); then
            COV_COLOR="${GREEN}"
        elif (( $(echo "$AVG_COV >= 30" | bc -l) )); then
            COV_COLOR="${YELLOW}"
        else
            COV_COLOR="${RED}"
        fi
        
        echo -e "${BLUE}├─────────────────────────────────────────────┤${NC}"
        printf "${BLUE}│${NC} %-30s ${COV_COLOR}%5.1f%%${NC} ${BLUE}│${NC}\n" "OVERALL AVERAGE COVERAGE:" "$AVG_COV"
    else
        echo -e "${BLUE}├─────────────────────────────────────────────┤${NC}"
        echo -e "${BLUE}│${NC} ${RED}No coverage data available - add tests!${NC}        ${BLUE}│${NC}"
    fi
    
    echo -e "${BLUE}└─────────────────────────────────────────────┘${NC}"
    
    # Add a warning for low coverage
    if [ $PACKAGES_WITH_REAL_COV -gt 0 ] && (( $(echo "$AVG_COV < 50" | bc -l) )); then
        echo -e "${YELLOW}⚠️  Warning: Test coverage is below 50%. Consider adding more tests.${NC}"
        echo -e "${YELLOW}   Good tests verify both expected behaviors and edge cases.${NC}"
    fi
    
    # Show test count summary
    echo -e "${BLUE}┌─ Test Summary ─────────────────────────────────┐${NC}"
    echo -e "${BLUE}│${NC} Total packages tested: $PACKAGE_COUNT"
    echo -e "${BLUE}│${NC} Packages with coverage: $PACKAGES_WITH_REAL_COV/$PACKAGE_COUNT"
    echo -e "${BLUE}└─────────────────────────────────────────────┘${NC}"
    
    # Check if tests are likely placeholder tests with no coverage
    if [ $PACKAGES_WITH_REAL_COV -eq 0 ]; then
        echo -e "${YELLOW}⚠️  Warning: Tests appear to be placeholder tests with no actual coverage.${NC}"
        echo -e "${YELLOW}    Consider implementing real unit tests for the project.${NC}"
    fi
    
    # If verbose is enabled, show the full test output
    if [ "$VERBOSE" = true ]; then
        echo -e "${BLUE}┌─ Detailed Test Output ───────────────────────┐${NC}"
        cat "$TEST_OUTPUT"
        echo -e "${BLUE}└─────────────────────────────────────────────┘${NC}"
    fi
    
    rm -f "$TEST_OUTPUT"
    
    # Add a visual separator to distinguish test results from build output
    echo -e "${BLUE}═════════════════════════════════════════════════${NC}"
}

# Function to build for the current platform
build_current_platform() {
    echo -e "${YELLOW}Building AI Commit Message Generator for current platform...${NC}"
    
    # Clean any previous build
    if [ -f "ai-commit-msg" ]; then
        if [ "$VERBOSE" = true ]; then
            echo "Removing previous build..."
        fi
        rm ai-commit-msg
    fi
    
    # Build the tool with error capture
    if [ "$VERBOSE" = true ]; then
        echo "Compiling with Go... (Version: $VERSION)"
    fi
    
    # Temporary file for errors
    ERROR_OUTPUT=$(mktemp)
    
    if ! go build -ldflags "-X main.version=$VERSION" -o ai-commit-msg ./cmd/ai-commit-msg 2>"$ERROR_OUTPUT"; then
        echo -e "${RED}Build failed!${NC}"
        cat "$ERROR_OUTPUT"
        rm "$ERROR_OUTPUT"
        exit 1
    fi
    
    rm "$ERROR_OUTPUT"
    
    # Make executable
    chmod +x ai-commit-msg
    
    # Check if build was successful
    if [ -f "ai-commit-msg" ]; then
        echo -e "${GREEN}Build successful! Created executable 'ai-commit-msg'${NC}"
    else
        echo -e "${RED}Build failed - executable not created!${NC}"
        exit 1
    fi
}

# Function to build for both macOS and Windows
build_cross_platform() {
    echo -e "${YELLOW}Building AI Commit Message Generator for multiple platforms...${NC}"
    
    # Create a directory for binaries if it doesn't exist
    mkdir -p bin
    
    # Function to build for a specific platform
    build_for_platform() {
        local os=$1
        local arch=$2
        local suffix=$3
        
        # Build message
        echo -ne "Building for ${os} (${arch})..."
        
        # Temporary file for errors
        ERROR_OUTPUT=$(mktemp)
        
        # Build command with error capture
        if GOOS=${os} GOARCH=${arch} go build -ldflags "-X main.version=$VERSION" -o bin/ai-commit-msg-${suffix} ./cmd/ai-commit-msg 2>"$ERROR_OUTPUT"; then
            echo -e " ${GREEN}✓${NC}"
        else
            echo -e " ${RED}FAILED${NC}"
            echo "Error building for ${os}/${arch}:"
            cat "$ERROR_OUTPUT"
            echo ""
        fi
        
        rm "$ERROR_OUTPUT"
    }
    
    # Build for each supported platform
    build_for_platform "darwin" "amd64" "darwin-amd64"
    build_for_platform "darwin" "arm64" "darwin-arm64"
    build_for_platform "windows" "amd64" "windows-amd64.exe"
    
    # Also build for Linux (AMD64) if we're on Linux
    if [ "$(uname)" = "Linux" ]; then
        build_for_platform "linux" "amd64" "linux-amd64"
    fi
    
    echo -e "${GREEN}Cross-compilation complete!${NC}"
    
    # Count successful builds
    BUILD_COUNT=$(ls -1 bin/ | wc -l | tr -d ' ')
    
    if [ "$VERBOSE" = true ]; then
        echo "Binaries available in the bin/ directory:"
        ls -la bin/
    else
        echo "Successfully built $BUILD_COUNT platform-specific binaries in the bin/ directory."
    fi
    
    # Create a copy of the appropriate binary for the current platform
    # (using copy instead of symlink for better Windows compatibility)
    OS="$(uname)"
    ARCH="$(uname -m 2>/dev/null || echo 'unknown')"
    
    if [ "$OS" = "Darwin" ]; then
        if [[ "$ARCH" == "arm64" ]]; then
            echo "Creating executable for macOS ARM64..."
            cp -f bin/ai-commit-msg-darwin-arm64 ai-commit-msg
        else
            echo "Creating executable for macOS AMD64..."
            cp -f bin/ai-commit-msg-darwin-amd64 ai-commit-msg
        fi
        chmod +x ai-commit-msg
    elif [[ "$OS" =~ MINGW|MSYS|CYGWIN ]]; then
        echo "Creating executable for Windows..."
        cp -f bin/ai-commit-msg-windows-amd64.exe ai-commit-msg.exe
    elif [ "$OS" = "Linux" ]; then
        echo "Creating executable for Linux..."
        cp -f bin/ai-commit-msg-linux-amd64 ai-commit-msg
        chmod +x ai-commit-msg
    fi
    
    echo -e "${GREEN}✅ Platform-specific executable created in the project root.${NC}"
}

# Check if code compiles first (always do this)
check_compilation

# Run tests if requested
if [ "$RUN_TESTS" = true ]; then
    # Show message about default test execution if this is a default run with no args
    if [ "$NO_ARGS" = true ]; then
        echo -e "${BLUE}Running tests as part of default build process...${NC}"
        echo -e "${BLUE}(Use --skip-tests flag to bypass testing)${NC}"
    fi
    run_tests
fi

# Exit if only testing was requested
if [ "$TEST_ONLY" = true ]; then
    echo -e "${GREEN}Testing completed. Exiting without building.${NC}"
    exit 0
fi

# Build the project
if [ "$CROSS_COMPILE" = true ]; then
    build_cross_platform
else
    build_current_platform
fi

# Get OS type for platform-specific instructions
OS="$(uname)"
ARCH="$(uname -m 2>/dev/null || echo 'unknown')"

# Create a bin directory for the release binaries if it doesn't exist
mkdir -p bin

# Copy the appropriate binary to the root directory for easy access
if [ "$OS" = "Darwin" ]; then
    if [[ "$ARCH" == "arm64" ]]; then
        echo "Detected macOS on Apple Silicon (ARM64)"
        cp -f bin/ai-commit-msg-darwin-arm64 ai-commit-msg
    else
        echo "Detected macOS on Intel (AMD64)"
        cp -f bin/ai-commit-msg-darwin-amd64 ai-commit-msg
    fi
    chmod +x ai-commit-msg
    echo -e "${GREEN}Created executable 'ai-commit-msg' in the current directory${NC}"
elif [[ "$OS" =~ MINGW|MSYS|CYGWIN ]]; then
    echo "Detected Windows"
    cp -f bin/ai-commit-msg-windows-amd64.exe ai-commit-msg.exe
    echo -e "${GREEN}Created executable 'ai-commit-msg.exe' in the current directory${NC}"
elif [ "$OS" = "Linux" ]; then
    # For Linux, we didn't build a dedicated binary in cross-platform, so use the default one
    echo "Detected Linux"
    cp -f ai-commit-msg bin/ai-commit-msg-linux
    echo -e "${GREEN}Created executable 'ai-commit-msg' in the current directory${NC}"
fi

# Optionally create a symlink to make it available in PATH
if [ "$OS" = "Darwin" ] || [ "$OS" = "Linux" ]; then
    if [ "$CREATE_SYMLINK" = true ]; then
        echo -e "${YELLOW}Creating symlink in /usr/local/bin...${NC}"
        sudo ln -sf "$SCRIPT_DIR/ai-commit-msg" /usr/local/bin/ai-commit-msg
        echo -e "${GREEN}Symlink created! You can now run 'ai-commit-msg' from anywhere.${NC}"
    else
        echo -e "${YELLOW}Tip: Use --symlink flag to create a symlink in /usr/local/bin${NC}"
        echo -e "     This would allow you to run 'ai-commit-msg' from anywhere."
    fi
elif [[ "$OS" =~ MINGW|MSYS|CYGWIN ]]; then
    echo -e "${YELLOW}Note: On Windows, you may want to add this directory to your PATH to run ai-commit-msg from anywhere.${NC}"
    echo "Alternatively, you can move the ai-commit-msg.exe file to a directory in your PATH."
fi

echo ""
echo -e "${BLUE}╔════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║               BUILD INFORMATION                 ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════╝${NC}"

if [ "$SKIP_TESTS" = true ]; then
  echo -e "${YELLOW}▶ Unit tests were skipped for this build${NC}"
  echo "  Use './build.sh --test-only' to run tests separately"
else
  echo -e "${GREEN}▶ All unit tests were executed as part of this build process${NC}"
  echo "  Check the test results summary above for details"
fi

if [ "$CROSS_COMPILE" = true ]; then
  echo -e "${GREEN}▶ Cross-platform binaries are available in the bin/ directory${NC}"
  echo "  - macOS Intel:     bin/ai-commit-msg-darwin-amd64"
  echo "  - macOS Apple:     bin/ai-commit-msg-darwin-arm64"
  echo "  - Windows:         bin/ai-commit-msg-windows-amd64.exe"
else
  echo -e "${GREEN}▶ Single platform binary built for your current system${NC}"
fi

echo -e "${GREEN}▶ A platform-specific executable is available in the project root${NC}"
if [ "$OS" = "Darwin" ]; then
  if [[ "$ARCH" == "arm64" ]]; then
    echo "  - ./ai-commit-msg (macOS ARM64 binary)"
  else
    echo "  - ./ai-commit-msg (macOS AMD64 binary)"
  fi
elif [[ "$OS" =~ MINGW|MSYS|CYGWIN ]]; then
  echo "  - ./ai-commit-msg.exe (Windows AMD64 binary)"
else
  echo "  - ./ai-commit-msg (Linux AMD64 binary)"
fi
echo ""

echo -e "${GREEN}Done!${NC}"