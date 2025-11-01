#include <array>
#include <string_view>
#include <optional>
#include <string>

enum class ConfigType {
    Dev,
    Prod
};

class Config {
public:
    static inline constexpr std::string_view kEnvVarName = "ACTIVE";
    static inline constexpr auto kDefaultConfigType = ConfigType::Dev;
    static inline constexpr std::string_view kDefaultFilename = "config.dev.yaml";
    [[nodiscard]] static std::string GetConfigFilename();

private:
    static inline constexpr std::array<std::pair<std::string_view, ConfigType>, 2> kConfigTypeMap {{
        {"DEV", ConfigType::Dev},
        {"PROD", ConfigType::Prod}
    }};

    static inline constexpr std::array<std::pair<ConfigType, std::string_view>, 2> kConfigFileNameMap {{
        {ConfigType::Dev, "config.dev.yaml"},
        {ConfigType::Prod, "config.prod.yaml"}
    }};

    [[nodiscard]] static constexpr std::optional<ConfigType> FromString(std::string_view name) noexcept {
        for (auto [key, value] : kConfigTypeMap) {
            if (key == name)
                return value;
        }
        return std::nullopt;
    }

    [[nodiscard]] static constexpr std::optional<std::string_view> GetFilename(ConfigType type) noexcept {
        for (auto [key, value] : kConfigFileNameMap) {
            if (key == type)
                return value;
        }
        return std::nullopt;
    }

    [[nodiscard]] static constexpr std::optional<std::string_view> FilenameFromString(std::string_view name) noexcept {
        if (auto t = FromString(name)) {
            return GetFilename(*t);
        }
        return std::nullopt;
    }
};
