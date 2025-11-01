// Implementation of Config::GetConfigFilename
#include "config.h"
#include <cstdlib>    // std::getenv
#include <algorithm>  // std::transform
#include <cctype>     // std::toupper
#include <string>     // std::string
#include <stdexcept>  // std::runtime_error

std::string Config::GetConfigFilename() {
    const char* env_val = std::getenv(Config::kEnvVarName.data());
    if (!env_val || !*env_val) {
        return std::string(Config::kDefaultFilename);
    }

    std::string env_str(env_val);
    std::transform(env_str.begin(), env_str.end(), env_str.begin(),
                   [](unsigned char c) { return static_cast<char>(std::toupper(c)); });

    if (auto type = FromString(env_str)) {
        if (auto filename = GetFilename(*type)) {
            return std::string(*filename);
        }
    }

    throw std::runtime_error("invalid config type");
}