include(cmake/SystemLink.cmake)
include(cmake/LibFuzzer.cmake)
include(CMakeDependentOption)
include(CheckCXXCompilerFlag)


include(CheckCXXSourceCompiles)


macro(scaffold_supports_sanitizers)
  if((CMAKE_CXX_COMPILER_ID MATCHES ".*Clang.*" OR CMAKE_CXX_COMPILER_ID MATCHES ".*GNU.*") AND NOT WIN32)

    message(STATUS "Sanity checking UndefinedBehaviorSanitizer, it should be supported on this platform")
    set(TEST_PROGRAM "int main() { return 0; }")

    # Check if UndefinedBehaviorSanitizer works at link time
    set(CMAKE_REQUIRED_FLAGS "-fsanitize=undefined")
    set(CMAKE_REQUIRED_LINK_OPTIONS "-fsanitize=undefined")
    check_cxx_source_compiles("${TEST_PROGRAM}" HAS_UBSAN_LINK_SUPPORT)

    if(HAS_UBSAN_LINK_SUPPORT)
      message(STATUS "UndefinedBehaviorSanitizer is supported at both compile and link time.")
      set(SUPPORTS_UBSAN ON)
    else()
      message(WARNING "UndefinedBehaviorSanitizer is NOT supported at link time.")
      set(SUPPORTS_UBSAN OFF)
    endif()
  else()
    set(SUPPORTS_UBSAN OFF)
  endif()

  if((CMAKE_CXX_COMPILER_ID MATCHES ".*Clang.*" OR CMAKE_CXX_COMPILER_ID MATCHES ".*GNU.*") AND WIN32)
    set(SUPPORTS_ASAN OFF)
  else()
    if (NOT WIN32)
      message(STATUS "Sanity checking AddressSanitizer, it should be supported on this platform")
      set(TEST_PROGRAM "int main() { return 0; }")

      # Check if AddressSanitizer works at link time
      set(CMAKE_REQUIRED_FLAGS "-fsanitize=address")
      set(CMAKE_REQUIRED_LINK_OPTIONS "-fsanitize=address")
      check_cxx_source_compiles("${TEST_PROGRAM}" HAS_ASAN_LINK_SUPPORT)

      if(HAS_ASAN_LINK_SUPPORT)
        message(STATUS "AddressSanitizer is supported at both compile and link time.")
        set(SUPPORTS_ASAN ON)
      else()
        message(WARNING "AddressSanitizer is NOT supported at link time.")
        set(SUPPORTS_ASAN OFF)
      endif()
    else()
      set(SUPPORTS_ASAN ON)
    endif()
  endif()
endmacro()

macro(scaffold_setup_options)
  option(scaffold_ENABLE_HARDENING "Enable hardening" ON)
  option(scaffold_ENABLE_COVERAGE "Enable coverage reporting" OFF)
  cmake_dependent_option(
    scaffold_ENABLE_GLOBAL_HARDENING
    "Attempt to push hardening options to built dependencies"
    ON
    scaffold_ENABLE_HARDENING
    OFF)

  scaffold_supports_sanitizers()

  if(NOT PROJECT_IS_TOP_LEVEL OR scaffold_PACKAGING_MAINTAINER_MODE)
    option(scaffold_ENABLE_IPO "Enable IPO/LTO" OFF)
    option(scaffold_WARNINGS_AS_ERRORS "Treat Warnings As Errors" OFF)
    option(scaffold_ENABLE_USER_LINKER "Enable user-selected linker" OFF)
    option(scaffold_ENABLE_SANITIZER_ADDRESS "Enable address sanitizer" OFF)
    option(scaffold_ENABLE_SANITIZER_LEAK "Enable leak sanitizer" OFF)
    option(scaffold_ENABLE_SANITIZER_UNDEFINED "Enable undefined sanitizer" OFF)
    option(scaffold_ENABLE_SANITIZER_THREAD "Enable thread sanitizer" OFF)
    option(scaffold_ENABLE_SANITIZER_MEMORY "Enable memory sanitizer" OFF)
    option(scaffold_ENABLE_UNITY_BUILD "Enable unity builds" OFF)
    option(scaffold_ENABLE_CLANG_TIDY "Enable clang-tidy" OFF)
    option(scaffold_ENABLE_CPPCHECK "Enable cpp-check analysis" OFF)
    option(scaffold_ENABLE_PCH "Enable precompiled headers" OFF)
    option(scaffold_ENABLE_CACHE "Enable ccache" OFF)
  else()
    option(scaffold_ENABLE_IPO "Enable IPO/LTO" ON)
    option(scaffold_WARNINGS_AS_ERRORS "Treat Warnings As Errors" ON)
    option(scaffold_ENABLE_USER_LINKER "Enable user-selected linker" OFF)
    option(scaffold_ENABLE_SANITIZER_ADDRESS "Enable address sanitizer" ${SUPPORTS_ASAN})
    option(scaffold_ENABLE_SANITIZER_LEAK "Enable leak sanitizer" OFF)
    option(scaffold_ENABLE_SANITIZER_UNDEFINED "Enable undefined sanitizer" ${SUPPORTS_UBSAN})
    option(scaffold_ENABLE_SANITIZER_THREAD "Enable thread sanitizer" OFF)
    option(scaffold_ENABLE_SANITIZER_MEMORY "Enable memory sanitizer" OFF)
    option(scaffold_ENABLE_UNITY_BUILD "Enable unity builds" OFF)
    option(scaffold_ENABLE_CLANG_TIDY "Enable clang-tidy" ON)
    option(scaffold_ENABLE_CPPCHECK "Enable cpp-check analysis" ON)
    option(scaffold_ENABLE_PCH "Enable precompiled headers" OFF)
    option(scaffold_ENABLE_CACHE "Enable ccache" ON)
  endif()

  if(NOT PROJECT_IS_TOP_LEVEL)
    mark_as_advanced(
      scaffold_ENABLE_IPO
      scaffold_WARNINGS_AS_ERRORS
      scaffold_ENABLE_USER_LINKER
      scaffold_ENABLE_SANITIZER_ADDRESS
      scaffold_ENABLE_SANITIZER_LEAK
      scaffold_ENABLE_SANITIZER_UNDEFINED
      scaffold_ENABLE_SANITIZER_THREAD
      scaffold_ENABLE_SANITIZER_MEMORY
      scaffold_ENABLE_UNITY_BUILD
      scaffold_ENABLE_CLANG_TIDY
      scaffold_ENABLE_CPPCHECK
      scaffold_ENABLE_COVERAGE
      scaffold_ENABLE_PCH
      scaffold_ENABLE_CACHE)
  endif()

  scaffold_check_libfuzzer_support(LIBFUZZER_SUPPORTED)
  if(LIBFUZZER_SUPPORTED AND (scaffold_ENABLE_SANITIZER_ADDRESS OR scaffold_ENABLE_SANITIZER_THREAD OR scaffold_ENABLE_SANITIZER_UNDEFINED))
    set(DEFAULT_FUZZER ON)
  else()
    set(DEFAULT_FUZZER OFF)
  endif()

  option(scaffold_BUILD_FUZZ_TESTS "Enable fuzz testing executable" ${DEFAULT_FUZZER})

endmacro()

macro(scaffold_global_options)
  if(scaffold_ENABLE_IPO)
    include(cmake/InterproceduralOptimization.cmake)
    scaffold_enable_ipo()
  endif()

  scaffold_supports_sanitizers()

  if(scaffold_ENABLE_HARDENING AND scaffold_ENABLE_GLOBAL_HARDENING)
    include(cmake/Hardening.cmake)
    if(NOT SUPPORTS_UBSAN 
       OR scaffold_ENABLE_SANITIZER_UNDEFINED
       OR scaffold_ENABLE_SANITIZER_ADDRESS
       OR scaffold_ENABLE_SANITIZER_THREAD
       OR scaffold_ENABLE_SANITIZER_LEAK)
      set(ENABLE_UBSAN_MINIMAL_RUNTIME FALSE)
    else()
      set(ENABLE_UBSAN_MINIMAL_RUNTIME TRUE)
    endif()
    message("${scaffold_ENABLE_HARDENING} ${ENABLE_UBSAN_MINIMAL_RUNTIME} ${scaffold_ENABLE_SANITIZER_UNDEFINED}")
    scaffold_enable_hardening(scaffold_options ON ${ENABLE_UBSAN_MINIMAL_RUNTIME})
  endif()
endmacro()

macro(scaffold_local_options)
  if(PROJECT_IS_TOP_LEVEL)
    include(cmake/StandardProjectSettings.cmake)
  endif()

  add_library(scaffold_warnings INTERFACE)
  add_library(scaffold_options INTERFACE)

  include(cmake/CompilerWarnings.cmake)
  scaffold_set_project_warnings(
    scaffold_warnings
    ${scaffold_WARNINGS_AS_ERRORS}
    ""
    ""
    ""
    "")

  if(scaffold_ENABLE_USER_LINKER)
    include(cmake/Linker.cmake)
    scaffold_configure_linker(scaffold_options)
  endif()

  include(cmake/Sanitizers.cmake)
  scaffold_enable_sanitizers(
    scaffold_options
    ${scaffold_ENABLE_SANITIZER_ADDRESS}
    ${scaffold_ENABLE_SANITIZER_LEAK}
    ${scaffold_ENABLE_SANITIZER_UNDEFINED}
    ${scaffold_ENABLE_SANITIZER_THREAD}
    ${scaffold_ENABLE_SANITIZER_MEMORY})

  set_target_properties(scaffold_options PROPERTIES UNITY_BUILD ${scaffold_ENABLE_UNITY_BUILD})

  if(scaffold_ENABLE_PCH)
    target_precompile_headers(
      scaffold_options
      INTERFACE
      <vector>
      <string>
      <utility>)
  endif()

  if(scaffold_ENABLE_CACHE)
    include(cmake/Cache.cmake)
    scaffold_enable_cache()
  endif()

  include(cmake/StaticAnalyzers.cmake)
  if(scaffold_ENABLE_CLANG_TIDY)
    scaffold_enable_clang_tidy(scaffold_options ${scaffold_WARNINGS_AS_ERRORS})
  endif()

  if(scaffold_ENABLE_CPPCHECK)
    scaffold_enable_cppcheck(${scaffold_WARNINGS_AS_ERRORS} "" # override cppcheck options
    )
  endif()

  if(scaffold_ENABLE_COVERAGE)
    include(cmake/Tests.cmake)
    scaffold_enable_coverage(scaffold_options)
  endif()

  if(scaffold_WARNINGS_AS_ERRORS)
    check_cxx_compiler_flag("-Wl,--fatal-warnings" LINKER_FATAL_WARNINGS)
    if(LINKER_FATAL_WARNINGS)
      # This is not working consistently, so disabling for now
      # target_link_options(scaffold_options INTERFACE -Wl,--fatal-warnings)
    endif()
  endif()

  if(scaffold_ENABLE_HARDENING AND NOT scaffold_ENABLE_GLOBAL_HARDENING)
    include(cmake/Hardening.cmake)
    if(NOT SUPPORTS_UBSAN 
       OR scaffold_ENABLE_SANITIZER_UNDEFINED
       OR scaffold_ENABLE_SANITIZER_ADDRESS
       OR scaffold_ENABLE_SANITIZER_THREAD
       OR scaffold_ENABLE_SANITIZER_LEAK)
      set(ENABLE_UBSAN_MINIMAL_RUNTIME FALSE)
    else()
      set(ENABLE_UBSAN_MINIMAL_RUNTIME TRUE)
    endif()
    scaffold_enable_hardening(scaffold_options OFF ${ENABLE_UBSAN_MINIMAL_RUNTIME})
  endif()

endmacro()
