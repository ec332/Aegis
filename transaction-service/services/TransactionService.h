#pragma once
#include <drogon/drogon.h>
#include <json/json.h>
#include <vector>
#include "repositories/TransactionRepository.h"
#include "models/Transactions.h"

class TransactionService {
public:
  explicit TransactionService(const drogon::orm::DbClientPtr& client)
    : repo_(client) {}

  drogon::Task<std::vector<drogon_model::transaction::Transactions>> findAll();
  drogon::Task<drogon_model::transaction::Transactions> findById(const std::string& id);
  drogon::Task<drogon_model::transaction::Transactions> create(const Json::Value& json);
  drogon::Task<drogon_model::transaction::Transactions> update(const std::string& id, const Json::Value& json);
  drogon::Task<size_t> deleteById(const std::string& id);
private:
  TransactionRepository repo_;
};