include(cmake/CPM.cmake)

# Done as a function so that updates to variables like
# CMAKE_CXX_FLAGS don't propagate out to other
# targets
function(scaffold_setup_dependencies)

    # For each dependency, see if it's
    # already been provided to us by a parent project

    # Prefer the canonical fmt target name and fall back if needed
    if(NOT TARGET fmt::fmt AND NOT TARGET fmtlib::fmtlib)
        cpmaddpackage(
          NAME fmt
          GITHUB_REPOSITORY "fmtlib/fmt"
          GIT_TAG 11.1.4
          OPTIONS "FMT_HEADER_ONLY ON"
        )
    endif()

    # Ensure fmt compiled sources have stdlib declarations even under strict include removal
    if(TARGET fmt)
        target_compile_options(fmt PRIVATE $<$<OR:$<CXX_COMPILER_ID:AppleClang>,$<CXX_COMPILER_ID:Clang>>:-include stdlib.h>)
    endif()

    if(NOT TARGET spdlog::spdlog)
        cpmaddpackage(
      NAME
      spdlog
      VERSION
      1.15.2
      GITHUB_REPOSITORY
      "gabime/spdlog"
      OPTIONS
      "SPDLOG_FMT_EXTERNAL ON")
    endif()

    if(BUILD_TESTING)
        if(NOT TARGET Catch2::Catch2WithMain)
            cpmaddpackage("gh:catchorg/Catch2@3.8.1")
        endif()
    endif()

    if(NOT TARGET tools::tools)
        cpmaddpackage("gh:lefticus/tools#update_build_system")
    endif()

endfunction()
