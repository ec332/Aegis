include(cmake/CPM.cmake)

# Done as a function so that updates to variables like
# CMAKE_CXX_FLAGS don't propagate out to other
# targets
function(scaffold_setup_dependencies)

    # For each dependency, see if it's
    # already been provided to us by a parent project

    if(APPLE)
        # Hint Homebrew OpenSSL locations if OPENSSL_ROOT_DIR isn't set
        if(NOT DEFINED OPENSSL_ROOT_DIR)
            if(EXISTS "/opt/homebrew/opt/openssl@3")
                set(OPENSSL_ROOT_DIR "/opt/homebrew/opt/openssl@3")
            elseif(EXISTS "/usr/local/opt/openssl@3")
                set(OPENSSL_ROOT_DIR "/usr/local/opt/openssl@3")
            endif()
        endif()
    endif()
    find_package(OpenSSL QUIET)
    set(_drogon_use_openssl OFF)
    if(OpenSSL_FOUND)
        set(_drogon_use_openssl ON)
    endif()

    if(NOT TARGET drogon)
        cpmaddpackage(
            NAME drogon
            VERSION 1.7.5
            GITHUB_REPOSITORY "drogonframework/drogon"
            GIT_TAG v1.7.5
            OPTIONS
                "BUILD_EXAMPLES OFF"
                "BUILD_CTL OFF"
                "BUILD_TESTING OFF"
                "USE_OPENSSL ${_drogon_use_openssl}"
        )
    endif()

    # Propagate Drogon include paths and link settings to consumers
    # (Any target linking scaffold_options will automatically get Drogon)
    target_compile_features(scaffold_options INTERFACE cxx_std_20)
    target_link_libraries(scaffold_options INTERFACE drogon)

    if(OpenSSL_FOUND)
        target_link_libraries(scaffold_options INTERFACE OpenSSL::SSL OpenSSL::Crypto)
    endif()

    # Some platforms may require pthread; Drogon transitively handles this,
    # but make it explicit for robustness on UNIX
    if(UNIX AND NOT APPLE)
        find_package(Threads REQUIRED)
        target_link_libraries(scaffold_options INTERFACE Threads::Threads)
    endif()

    if(NOT TARGET fmtlib::fmtlib)
        cpmaddpackage("gh:fmtlib/fmt#11.1.4")
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
