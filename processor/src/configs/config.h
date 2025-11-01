#include <string>

#ifndef CONFIG_H
#define CONFIG_H

struct Config {
    int database_port_;
    std::string database_host_;
    std::string database_name_;
};


#endif
