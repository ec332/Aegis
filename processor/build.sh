#!/bin/bash
set -e
chmod +x build.sh
if ! command -v cmake >/dev/null 2>&1; then
    echo "CMake not found."

    # Check for Homebrew
    if ! command -v brew >/dev/null 2>&1; then
        echo "Homebrew not found. Please install Homebrew first: https://brew.sh/"
        exit 1
    fi

    echo "Installing CMake via Homebrew..."
    brew install cmake
    echo "✅ CMake installed successfully."
fi

# Ensure analysis tools are available
if ! command -v brew >/dev/null 2>&1; then
    echo "Homebrew not found. Please install Homebrew first: https://brew.sh/"
    exit 1
fi

# Install LLVM for clang-tidy and better sanitizer support (if missing)
if ! command -v clang-tidy >/dev/null 2>&1; then
    echo "Installing LLVM (clang/clang-tidy) via Homebrew..."
    brew install llvm
fi

# Install cppcheck (if missing)
if ! command -v cppcheck >/dev/null 2>&1; then
    echo "Installing cppcheck via Homebrew..."
    brew install cppcheck
fi

# Require mode argument (fast | nofast)
if [ $# -ne 1 ]; then
    echo "Usage: $0 fast|nofast"
    exit 1
fi

MODE="$1"
if [ "$MODE" != "fast" ] && [ "$MODE" != "nofast" ]; then
    echo "Invalid mode: $MODE"
    echo "Usage: $0 fast|nofast"
    exit 1
fi

# Prefer LLVM clang++ if available (needed for best sanitizer support)
LLVM_PREFIX=$(brew --prefix llvm 2>/dev/null || true)
LLVM_CLANGXX="$LLVM_PREFIX/bin/clang++"
if [ -x "$LLVM_CLANGXX" ]; then
    echo "Using LLVM clang++ at $LLVM_CLANGXX"
    export PATH="$LLVM_PREFIX/bin:$PATH"
fi

# Build the project with requested defaults
echo "Building project with defaults: Hardening, ASAN, UBSAN, clang-tidy, cppcheck..."
CMAKE_ARGS=(
  -Dscaffold_ENABLE_HARDENING=ON
  -Dscaffold_ENABLE_GLOBAL_HARDENING=ON
  -Dscaffold_ENABLE_SANITIZER_ADDRESS=ON
  -Dscaffold_ENABLE_SANITIZER_UNDEFINED=ON
  -Dscaffold_ENABLE_CLANG_TIDY=ON
  -Dscaffold_ENABLE_CPPCHECK=ON
)

# FAST mode: disable expensive checks for quick iteration
if [ "$MODE" = "fast" ]; then
  echo "FAST mode enabled: Disabling sanitizers, clang-tidy, cppcheck, and global hardening."
  CMAKE_ARGS=(
    -Dscaffold_ENABLE_HARDENING=OFF
    -Dscaffold_ENABLE_GLOBAL_HARDENING=OFF
    -Dscaffold_ENABLE_SANITIZER_ADDRESS=OFF
    -Dscaffold_ENABLE_SANITIZER_UNDEFINED=OFF
    -Dscaffold_ENABLE_CLANG_TIDY=OFF
    -Dscaffold_ENABLE_CPPCHECK=OFF
  )
fi

# If LLVM clang++ is available, use it
if [ -x "$LLVM_CLANGXX" ]; then
  CMAKE_ARGS+=(-DCMAKE_CXX_COMPILER="$LLVM_CLANGXX")
fi

cmake -S . -B build "${CMAKE_ARGS[@]}"
cmake --build build
echo "✅ Build complete!"
